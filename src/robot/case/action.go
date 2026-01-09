package atsf4g_go_robot_case

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	base "github.com/atframework/atsf4g-go/robot/base"
	utils "github.com/atframework/atsf4g-go/robot/utils"
	"github.com/atframework/libatapp-go"
)

type TaskActionCase struct {
	base.TaskActionBase
	Fn         func(*TaskActionCase, string) error
	logHandler func(openId string, format string, a ...any)
	OpenId     string
}

func (t *TaskActionCase) HookRun() error {
	return t.Fn(t, t.OpenId)
}

func (t *TaskActionCase) Log(format string, a ...any) {
	t.logHandler(t.OpenId, format, a...)
}

func init() {
	var _ base.TaskActionImpl = &TaskActionCase{}
	utils.RegisterCommand([]string{"run-case"}, CmdRunCase, "<case name> <openid-prefix> <user-count> <batch-count> <run-time>", "运行用例", AutoCompleteCaseName, 0)
}

type CaseAction struct {
	fun     func(*TaskActionCase, string) error
	timeout time.Duration
}

var caseMapContainer = make(map[string]CaseAction)

func RegisterCase(name string, fn func(*TaskActionCase, string) error, timeout time.Duration) {
	caseMapContainer[name] = CaseAction{
		fun:     fn,
		timeout: timeout,
	}
}

func AutoCompleteCaseName(string) []string {
	var res []string
	for k := range caseMapContainer {
		res = append(res, k)
	}
	return res
}

var (
	ProgressBarTotalCount   int64
	ProgressBarCurrentCount atomic.Int64

	FailedCount      atomic.Int64
	TotalFailedCount atomic.Int64

	RefreshCount int64
	RefreshFunc  *time.Timer
)

func RefreshProgressBar(first bool) {
	countProgressBar := ""
	RefreshCount++
	loadType := RefreshCount % 3
	{
		width := 25
		progress := float64(ProgressBarCurrentCount.Load()) / float64(ProgressBarTotalCount)
		completed := int(progress * float64(width))
		countProgressBar = fmt.Sprintf("[%-*s] %d/%d", width, strings.Repeat("#", completed), ProgressBarCurrentCount.Load(), ProgressBarTotalCount)
	}
	loadTypeString := ""
	switch loadType {
	case 0:
		loadTypeString = "--"
	case 1:
		loadTypeString = "\\"
	case 2:
		loadTypeString = "/"
	}
	if first {
		fmt.Printf("%s Total:%s || Failed:%d             ", loadTypeString, countProgressBar, FailedCount.Load())
	} else {
		fmt.Printf("\r%s Total:%s || Failed:%d             ", loadTypeString, countProgressBar, FailedCount.Load())
	}
	if ProgressBarCurrentCount.Load() >= ProgressBarTotalCount {
		return
	}
	RefreshFunc = time.AfterFunc(time.Second, func() { RefreshProgressBar(false) })
}

func InitProgressBar(totalCount int64) {
	ProgressBarTotalCount = totalCount
	RefreshCount = 0
	ProgressBarCurrentCount.Store(0)
	FailedCount.Store(0)

	RefreshProgressBar(true)
}

func AddProgressBarCount() {
	ProgressBarCurrentCount.Add(1)
}

func RunCaseFile(caseFile string) error {
	file, err := os.Open(caseFile)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		args := strings.Fields(line)
		if len(args) == 0 {
			continue
		}

		if errMsg := CmdRunCase(nil, args); len(errMsg) > 0 {
			return fmt.Errorf("run case failed: %s, args: %v", errMsg, args)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func CmdRunCase(_ base.TaskActionImpl, cmd []string) string {
	if len(cmd) < 5 {
		return "Args Error"
	}

	caseName := cmd[0]
	caseAction, ok := caseMapContainer[caseName]
	if !ok {
		return "Case Not Found"
	}

	openIdPrefix := cmd[1]
	if openIdPrefix == "" {
		return "OpenId Prefix Empty"
	}

	userCount, err := strconv.ParseInt(cmd[2], 10, 32)
	if err != nil {
		return err.Error()
	}

	batchCount, err := strconv.ParseInt(cmd[3], 10, 32)
	if err != nil {
		return err.Error()
	}
	if batchCount <= 0 {
		return "Batch Count Must Greater Than 0"
	}
	if batchCount > userCount {
		batchCount = userCount
	}

	runTime, err := strconv.ParseInt(cmd[4], 10, 32)
	if err != nil {
		return err.Error()
	}

	totalCount := atomic.Int64{}
	totalCount.Store(userCount * runTime)

	timeCounter := sync.Map{}
	openidChannel := make(chan string, userCount)
	for i := int64(0); i < userCount; i++ {
		// 初始化Time统计
		openId := openIdPrefix + strconv.FormatInt(i, 10)
		timeCounter.Store(openId, int32(runTime))
		// 初始化OpenId令牌
		openidChannel <- openId
	}

	InitProgressBar(totalCount.Load())

	caseActionChannel := make(chan *TaskActionCase, batchCount) // 限制并发数

	beginTime := time.Now()

	bufferWriter, _ := libatapp.NewLogBufferedRotatingWriter(nil,
		fmt.Sprintf("../log/%s.%s.%%N.log", caseName, beginTime.Format("15.04.05")), "", 50*1024*1024, 5, time.Second*3, 0)
	runtime.SetFinalizer(bufferWriter, func(writer *libatapp.LogBufferedRotatingWriter) {
		writer.Close()
	})

	for i := int64(0); i < batchCount; i++ {
		// 创建TaskActionCase
		task := &TaskActionCase{
			TaskActionBase: *base.NewTaskActionBase(caseAction.timeout, "Case Runner"),
			Fn:             caseAction.fun,
			logHandler: func(openId string, format string, a ...any) {
				logString := fmt.Sprintf("[%s][%s]: %s", time.Now().Format("2006-01-02 15:04:05.000"), openId, fmt.Sprintf(format, a...))
				bufferWriter.Write(lu.StringtoBytes(logString))
			},
		}
		task.TaskActionBase.Impl = task
		caseActionChannel <- task
		task.InitOnFinish(func(err error) {
			openId := task.OpenId
			currentCount, _ := timeCounter.Load(openId)
			currentCountInt := currentCount.(int32)
			timeCounter.Store(openId, currentCountInt-1)
			if currentCountInt-1 > 0 {
				// 还有运行次数，继续放回OpenId
				openidChannel <- openId
			}
			if err != nil {
				FailedCount.Add(1)
				TotalFailedCount.Add(1)
				task.Log("Run Case[%s] Failed: %v", task.Name, err)
			}
			caseActionChannel <- task
		})
	}

	mgr := base.NewTaskActionManager()
	finishChannel := make(chan struct{})
	go func() {
		successCount := int64(0)
		for action := range caseActionChannel {
			// 取出OpenId
			openId := <-openidChannel
			action.OpenId = openId
			// 运行TaskAction
			mgr.RunTaskAction(action)
			successCount++
			AddProgressBarCount()
			if successCount >= totalCount.Load() {
				break
			}
		}
		// 等待任务完成
		mgr.WaitAll()
		RefreshFunc.Stop()
		RefreshProgressBar(false)
		fmt.Printf("\n")
		finishChannel <- struct{}{}
	}()
	<-finishChannel
	if TotalFailedCount.Load() == 0 {
		utils.StdoutLog(fmt.Sprintf("Complete All Success Args: %v", cmd))
	} else {
		return fmt.Sprintf("Complete With %d Failed", TotalFailedCount.Load())
	}
	return ""
}

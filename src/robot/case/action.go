package atsf4g_go_robot_case

import (
	"fmt"
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

	ProgressBarTotalUser   int64
	ProgressBarCurrentUser atomic.Int64

	RefreshCount int64
	RefreshFunc  *time.Timer
)

func RefreshProgressBar(first bool) {
	countProgressBar := ""
	userProgressBar := ""
	RefreshCount++
	loadType := RefreshCount % 3
	{
		width := 25
		progress := float64(ProgressBarCurrentCount.Load()) / float64(ProgressBarTotalCount)
		completed := int(progress * float64(width))
		countProgressBar = fmt.Sprintf("[%-*s] %d/%d", width, strings.Repeat("#", completed), ProgressBarCurrentCount.Load(), ProgressBarTotalCount)
	}
	{
		width := 25
		progress := float64(ProgressBarCurrentUser.Load()) / float64(ProgressBarTotalUser)
		completed := int(progress * float64(width))
		userProgressBar = fmt.Sprintf("[%-*s] %d/%d", width, strings.Repeat("#", completed), ProgressBarCurrentUser.Load(), ProgressBarTotalUser)
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
		fmt.Printf("%s Total:%s || User:%s             ", loadTypeString, countProgressBar, userProgressBar)
	} else {
		fmt.Printf("\r%s Total:%s || User:%s             ", loadTypeString, countProgressBar, userProgressBar)
	}
	if ProgressBarCurrentCount.Load() >= ProgressBarTotalCount {
		fmt.Printf("\n")
		utils.StdoutLog("Complete")
		return
	}
	RefreshFunc = time.AfterFunc(time.Second, func() { RefreshProgressBar(false) })
}

func InitProgressBar(totalCount int64, totalUser int64) {
	ProgressBarTotalCount = totalCount
	ProgressBarTotalUser = totalUser
	RefreshCount = 0
	ProgressBarCurrentCount.Store(0)
	ProgressBarCurrentUser.Store(0)

	RefreshProgressBar(true)
}

func AddProgressBarCount() {
	ProgressBarCurrentCount.Add(1)
}

func AddProgressBarUser() {
	ProgressBarCurrentUser.Add(1)
}

func CmdRunCase(action base.TaskActionImpl, cmd []string) string {
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

	InitProgressBar(totalCount.Load(), userCount)

	caseActionChannel := make(chan *TaskActionCase, batchCount) // 限制并发数

	beginTime := time.Now()

	bufferWriter, _ := libatapp.NewLogBufferedRotatingWriter(nil,
		fmt.Sprintf("../log/%s.%s.%%N.log", caseName, beginTime.Format("15.04.05")), "", 50*1024*1024, 5, time.Second*3)
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
			} else {
				AddProgressBarUser()
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
		RefreshFunc.Stop()
		RefreshProgressBar(false)
		// 等待任务完成
		mgr.WaitAll()
		finishChannel <- struct{}{}
	}()
	<-finishChannel
	return ""
}

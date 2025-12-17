package atframework_component_uuid

import (
	"container/list"
	"fmt"
	"sync"
	"unsafe"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	config "github.com/atframework/atsf4g-go/component-config"
	db "github.com/atframework/atsf4g-go/component-db"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
)

type uniqueIDKey struct {
	majorType uint32
	minorType uint32
	pathType  uint32
}

type uniqueIDPool struct {
	mu sync.Mutex

	base  uint64 // DB Key
	index uint64 // Local Index

	ioTask          cd.TaskActionImpl
	awaitIOTaskList list.List
}

var (
	uniqueIDPools = map[uniqueIDKey]*uniqueIDPool{}
	uniqueIDMu    sync.RWMutex
)

func bitsOffForMajor(majorType private_protocol_pbdesc.EnGlobalUUIDMajorType) int64 {
	switch majorType {
	case private_protocol_pbdesc.EnGlobalUUIDMajorType_EN_GLOBAL_UUID_MAT_ACCOUNT_ID,
		private_protocol_pbdesc.EnGlobalUUIDMajorType_EN_GLOBAL_UUID_MAT_GUILD_ID:
		// POOL => 1 | * | 5
		// EN_GLOBAL_UUID_MAT_ACCOUNT_ID:  [1 | 55 | 5] | 3
		// EN_GLOBAL_UUID_MAT_GUILD_ID:    [1 | 55 | 5] | 3
		// 公会和玩家账号分配采用短ID模式
		return 5
	default:
		// POOL => 1 | 50 | 13
		return 13
	}
}

func getUniqueIDPool(key uniqueIDKey) *uniqueIDPool {
	uniqueIDMu.RLock()
	pool := uniqueIDPools[key]
	uniqueIDMu.RUnlock()
	if pool != nil {
		return pool
	}

	uniqueIDMu.Lock()
	defer uniqueIDMu.Unlock()
	pool = uniqueIDPools[key]
	if pool == nil {
		pool = &uniqueIDPool{}
		uniqueIDPools[key] = pool
	}
	return pool
}

func GenerateGlobalUniqueID(ctx cd.AwaitableContext,
	majorType private_protocol_pbdesc.EnGlobalUUIDMajorType,
	minorType private_protocol_pbdesc.EnGlobalUUIDMinorType,
	pathType private_protocol_pbdesc.EnGlobalUUIDPatchType) (uuid uint64, result cd.RpcResult) {
	pool := getUniqueIDPool(uniqueIDKey{majorType: uint32(majorType), minorType: uint32(minorType), pathType: uint32(pathType)})
	bitsOff := bitsOffForMajor(majorType)
	bitsRange := uint64(1) << bitsOff

	currentTask := ctx.GetAction()
	if lu.IsNil(currentTask) || currentTask.GetTaskId() == 0 {
		ctx.LogError("should in task")
		return 0, cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_RPC_NO_TASK)
	}

	const maxRetry = 5
	shouldWake := false
	for range maxRetry {
		if currentTask.IsExiting() {
			result = cd.CreateRpcResultError(fmt.Errorf("task exiting"), public_protocol_pbdesc.EnErrorCode_EN_ERR_TIMEOUT)
			break
		}

		pool.mu.Lock()
		if !lu.IsNil(pool.ioTask) && !pool.ioTask.IsExiting() && pool.ioTask != currentTask {
			// 等待当前任务完成IO
			cd.YieldTaskAction(ctx, currentTask, &cd.DispatcherAwaitOptions{
				Type:     uint64(uintptr(unsafe.Pointer(pool))),
				Sequence: currentTask.GetTaskId(),
				Timeout:  config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetTask().GetCsmsg().GetTimeout().AsDuration(),
			}, func() cd.RpcResult {
				pool.awaitIOTaskList.PushBack(currentTask)
				return cd.CreateRpcResultOk()
			}, func() {
				pool.mu.Unlock()
			})
			continue
		}

		// 这里开始没有任务在跑IO 并且拿到了锁
		if pool.base == 0 || pool.index >= bitsRange {
			// 需要从数据库分配新的区块
			pool.ioTask = currentTask
			shouldWake = true
			pool.mu.Unlock()

			val, ret := db.DatabaseTableUuidAllocatorAtomicIncMajorTypeMinorTypePathTypeAutoIncId(ctx,
				uint32(majorType), uint32(minorType), uint32(pathType), 1)
			if ret.IsError() {
				result = ret
				continue
			}

			pool.mu.Lock()
			pool.base = val
			pool.index = 0
		}

		uuid = (pool.base << bitsOff) | pool.index
		pool.index++
		pool.mu.Unlock()
		break
	}

	if shouldWake {
		// 唤醒等待的任务
		pool.mu.Lock()
		if pool.ioTask == currentTask {
			pool.ioTask = nil
			for pool.awaitIOTaskList.Len() > 0 {
				elem := pool.awaitIOTaskList.Front()
				task := elem.Value.(cd.TaskActionImpl)
				pool.awaitIOTaskList.Remove(elem)
				if task.IsExiting() {
					continue
				}
				cd.ResumeTaskAction(ctx, task, &cd.DispatcherResumeData{
					Message: &cd.DispatcherRawMessage{
						Type: uint64(uintptr(unsafe.Pointer(pool))),
					},
					Sequence: task.GetTaskId(),
				})
			}
		}
		pool.mu.Unlock()
	}

	if uuid == 0 && result.IsOK() {
		result = cd.CreateRpcResultError(fmt.Errorf("generate uuid failed"), public_protocol_pbdesc.EnErrorCode_EN_ERR_UNKNOWN)
	}
	return
}

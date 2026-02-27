package atframework_component_db

import (
	"fmt"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	cd "github.com/atframework/atsf4g-go/component-dispatcher"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
	"github.com/atframework/libatapp-go"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// ZAdd 选项枚举
// ============================================================================

// ZAddExistenceOption ZAdd 命令的存在性选项
type ZAddExistenceOption int32

const (
	// ZAddExistenceNone 不指定，使用默认行为（不存在则添加，存在则更新）
	ZAddExistenceNone ZAddExistenceOption = 0
	// ZAddExistenceNX 仅当成员不存在时才添加（不更新已存在的成员）
	ZAddExistenceNX ZAddExistenceOption = 1
	// ZAddExistenceXX 仅当成员已存在时才更新（不添加新成员）
	ZAddExistenceXX ZAddExistenceOption = 2
)

// ZAddComparisonOption ZAdd 命令的比较选项
type ZAddComparisonOption int32

const (
	// ZAddComparisonNone 不指定，直接设置新分数
	ZAddComparisonNone ZAddComparisonOption = 0
	// ZAddComparisonGT 仅当新分数大于当前分数时才更新（成员不存在时总是添加）
	ZAddComparisonGT ZAddComparisonOption = 1
	// ZAddComparisonLT 仅当新分数小于当前分数时才更新（成员不存在时总是添加）
	ZAddComparisonLT ZAddComparisonOption = 2
)

// ============================================================================
// ZAdd 相关结构与接口
// ============================================================================

// SortedSetMember ZAdd 操作的成员结构
type SortedSetMember struct {
	Member string  // 成员键（始终为 string 类型）
	Score  float64 // 分数
}

// ZAddIncrResult ZAdd INCR 模式下的返回结果
type ZAddIncrResult struct {
	NewScore float64 // INCR 模式下返回的新分数
	Exists   bool    // 在 NX/XX 模式下表示操作是否实际执行
}

// SortedSetZAdd 向有序集合添加成员（支持多个成员）
// existence: NX/XX 选项
// comparison: GT/LT 选项
func SortedSetZAdd(ctx cd.AwaitableContext, index string, tableName string,
	dispatcher *cd.RedisMessageDispatcher, instance *redis.ClusterClient,
	members []SortedSetMember,
	existence ZAddExistenceOption,
	comparison ZAddComparisonOption,
) (retResult cd.RpcResult) {
	awaitOption := dispatcher.CreateDispatcherAwaitOptions()
	currentAction := ctx.GetAction()
	if lu.IsNil(currentAction) {
		ctx.LogError("not in context action")
		retResult = cd.CreateRpcResultError(fmt.Errorf("action not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	if currentAction.GetRpcContext() == nil || lu.IsNil(currentAction.GetRpcContext().GetContext()) {
		ctx.LogError("not found context")
		retResult = cd.CreateRpcResultError(fmt.Errorf("context not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}

	// 构建 redis.Z 切片
	redisMembers := make([]redis.Z, 0, len(members))
	for _, m := range members {
		redisMembers = append(redisMembers, redis.Z{
			Score:  m.Score,
			Member: m.Member,
		})
	}

	type innerPrivateData struct{}

	pushActionFunc := func() cd.RpcResult {
		err := ctx.GetApp().PushAction(func(app_action *libatapp.AppActionData) error {
			ctx.GetApp().GetLogger(2).LogDebug("SortedSetZAdd Send", "TableName", tableName, "Seq", awaitOption.Sequence, "index", index, "memberCount", len(members))

			args := redis.ZAddArgs{
				Members: redisMembers,
			}
			switch existence {
			case ZAddExistenceNX:
				args.NX = true
			case ZAddExistenceXX:
				args.XX = true
			}
			switch comparison {
			case ZAddComparisonGT:
				args.GT = true
			case ZAddComparisonLT:
				args.LT = true
			}

			cmdResult, redisError := instance.ZAddArgs(ctx.GetContext(), index, args).Result()
			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}
			if redisError != nil {
				ctx.GetApp().GetLogger(2).LogError("SortedSetZAdd Recv Error", "TableName", tableName, "Seq", awaitOption.Sequence, "redisError", redisError)
				resumeData.Result = cd.CreateRpcResultError(redisError, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
				resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("zadd failed resume error", "TableName", tableName, "err", resumeError)
					return resumeError
				}
				return redisError
			}
			ctx.GetApp().GetLogger(2).LogDebug("SortedSetZAdd Recv Success", "TableName", tableName, "Seq", awaitOption.Sequence, "changed", cmdResult)
			resumeData.PrivateData = &innerPrivateData{}
			resumeData.Result = cd.CreateRpcResultOk()
			resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
			if resumeError != nil {
				ctx.LogError("zadd failed resume error", "TableName", tableName, "err", resumeError)
				return resumeError
			}
			return nil
		}, nil, nil)
		if err != nil {
			return cd.CreateRpcResultError(err, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		}
		return cd.CreateRpcResultOk()
	}
	var resumeData *cd.DispatcherResumeData
	resumeData, retResult = cd.YieldTaskAction(ctx, currentAction, awaitOption, pushActionFunc, nil)
	if retResult.IsError() {
		return
	}
	if resumeData.Result.IsError() {
		retResult = resumeData.Result
		return
	}
	privateData, ok := resumeData.PrivateData.(*innerPrivateData)
	if !ok {
		ctx.LogError("zadd PrivateData failed not innerPrivateData")
		retResult = cd.CreateRpcResultError(fmt.Errorf("private data not innerPrivateData"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	_ = privateData
	return
}

// SortedSetZAddIncr 向有序集合执行 ZADD INCR 操作（仅支持单个成员）
func SortedSetZAddIncr(ctx cd.AwaitableContext, index string, tableName string,
	dispatcher *cd.RedisMessageDispatcher, instance *redis.ClusterClient,
	member SortedSetMember,
) (result ZAddIncrResult, retResult cd.RpcResult) {
	awaitOption := dispatcher.CreateDispatcherAwaitOptions()
	currentAction := ctx.GetAction()
	if lu.IsNil(currentAction) {
		ctx.LogError("not in context action")
		retResult = cd.CreateRpcResultError(fmt.Errorf("action not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	if currentAction.GetRpcContext() == nil || lu.IsNil(currentAction.GetRpcContext().GetContext()) {
		ctx.LogError("not found context")
		retResult = cd.CreateRpcResultError(fmt.Errorf("context not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}

	type innerPrivateData struct {
		NewScore float64
		Exists   bool
	}

	pushActionFunc := func() cd.RpcResult {
		err := ctx.GetApp().PushAction(func(app_action *libatapp.AppActionData) error {
			ctx.GetApp().GetLogger(2).LogDebug("SortedSetZAddIncr Send", "TableName", tableName, "Seq", awaitOption.Sequence, "index", index, "member", member.Member, "score", member.Score)

			args := redis.ZAddArgs{
				Members: []redis.Z{{Score: member.Score, Member: member.Member}},
			}

			cmdResult, redisError := instance.ZAddArgsIncr(ctx.GetContext(), index, args).Result()
			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}
			if redisError != nil && redisError != redis.Nil {
				ctx.GetApp().GetLogger(2).LogError("SortedSetZAddIncr Recv Error", "TableName", tableName, "Seq", awaitOption.Sequence, "redisError", redisError)
				resumeData.Result = cd.CreateRpcResultError(redisError, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
				resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("zadd incr failed resume error", "TableName", tableName, "err", resumeError)
					return resumeError
				}
				return redisError
			}
			exists := redisError != redis.Nil
			ctx.GetApp().GetLogger(2).LogDebug("SortedSetZAddIncr Recv Success", "TableName", tableName, "Seq", awaitOption.Sequence, "newScore", cmdResult, "exists", exists)
			resumeData.PrivateData = &innerPrivateData{NewScore: cmdResult, Exists: exists}
			resumeData.Result = cd.CreateRpcResultOk()
			resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
			if resumeError != nil {
				ctx.LogError("zadd incr failed resume error", "TableName", tableName, "err", resumeError)
				return resumeError
			}
			return nil
		}, nil, nil)
		if err != nil {
			return cd.CreateRpcResultError(err, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		}
		return cd.CreateRpcResultOk()
	}
	var resumeData *cd.DispatcherResumeData
	resumeData, retResult = cd.YieldTaskAction(ctx, currentAction, awaitOption, pushActionFunc, nil)
	if retResult.IsError() {
		return
	}
	if resumeData.Result.IsError() {
		retResult = resumeData.Result
		return
	}
	privateData, ok := resumeData.PrivateData.(*innerPrivateData)
	if !ok {
		ctx.LogError("zadd incr PrivateData failed not innerPrivateData")
		retResult = cd.CreateRpcResultError(fmt.Errorf("private data not innerPrivateData"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	result.NewScore = privateData.NewScore
	result.Exists = privateData.Exists
	return
}

// ============================================================================
// ZRange 相关结构与接口
// ============================================================================

// SortedSetRangeMember ZRange 返回的成员结构（带 WITHSCORES）
type SortedSetRangeMember struct {
	Member string  // 成员键
	Score  float64 // 分数
}

// SortedSetZRangeByRank 使用默认模式（BYRANK）查询有序集合，固定带 WITHSCORES
// start/stop 为 rank 范围（0-based，支持负数）
func SortedSetZRangeByRank(ctx cd.AwaitableContext, index string, tableName string,
	dispatcher *cd.RedisMessageDispatcher, instance *redis.ClusterClient,
	start int64, stop int64, rev bool,
) (members []SortedSetRangeMember, retResult cd.RpcResult) {
	awaitOption := dispatcher.CreateDispatcherAwaitOptions()
	currentAction := ctx.GetAction()
	if lu.IsNil(currentAction) {
		ctx.LogError("not in context action")
		retResult = cd.CreateRpcResultError(fmt.Errorf("action not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	if currentAction.GetRpcContext() == nil || lu.IsNil(currentAction.GetRpcContext().GetContext()) {
		ctx.LogError("not found context")
		retResult = cd.CreateRpcResultError(fmt.Errorf("context not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}

	type innerPrivateData struct {
		Members []SortedSetRangeMember
	}

	pushActionFunc := func() cd.RpcResult {
		err := ctx.GetApp().PushAction(func(app_action *libatapp.AppActionData) error {
			ctx.GetApp().GetLogger(2).LogDebug("SortedSetZRangeByRank Send", "TableName", tableName, "Seq", awaitOption.Sequence, "index", index, "start", start, "stop", stop, "rev", rev)

			args := redis.ZRangeArgs{
				Key:   index,
				Start: start,
				Stop:  stop,
				Rev:   rev,
			}
			cmdResult, redisError := instance.ZRangeArgsWithScores(ctx.GetContext(), args).Result()
			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}
			if redisError != nil {
				ctx.GetApp().GetLogger(2).LogError("SortedSetZRangeByRank Recv Error", "TableName", tableName, "Seq", awaitOption.Sequence, "redisError", redisError)
				resumeData.Result = cd.CreateRpcResultError(redisError, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
				resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("zrange by rank failed resume error", "TableName", tableName, "err", resumeError)
					return resumeError
				}
				return redisError
			}
			resultMembers := make([]SortedSetRangeMember, 0, len(cmdResult))
			for _, z := range cmdResult {
				resultMembers = append(resultMembers, SortedSetRangeMember{
					Member: fmt.Sprintf("%v", z.Member),
					Score:  z.Score,
				})
			}
			ctx.GetApp().GetLogger(2).LogDebug("SortedSetZRangeByRank Recv Success", "TableName", tableName, "Seq", awaitOption.Sequence, "count", len(resultMembers))
			resumeData.PrivateData = &innerPrivateData{Members: resultMembers}
			resumeData.Result = cd.CreateRpcResultOk()
			resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
			if resumeError != nil {
				ctx.LogError("zrange by rank failed resume error", "TableName", tableName, "err", resumeError)
				return resumeError
			}
			return nil
		}, nil, nil)
		if err != nil {
			return cd.CreateRpcResultError(err, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		}
		return cd.CreateRpcResultOk()
	}
	var resumeData *cd.DispatcherResumeData
	resumeData, retResult = cd.YieldTaskAction(ctx, currentAction, awaitOption, pushActionFunc, nil)
	if retResult.IsError() {
		return
	}
	if resumeData.Result.IsError() {
		retResult = resumeData.Result
		return
	}
	privateData, ok := resumeData.PrivateData.(*innerPrivateData)
	if !ok {
		ctx.LogError("zrange by rank PrivateData failed not innerPrivateData")
		retResult = cd.CreateRpcResultError(fmt.Errorf("private data not innerPrivateData"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	members = privateData.Members
	return
}

// ScoreBoundType 分数边界类型
type ScoreBoundType int32

const (
	// ScoreBoundValue 使用具体分数值
	ScoreBoundValue ScoreBoundType = 0
	// ScoreBoundNegativeInfinity 负无穷大 -inf
	ScoreBoundNegativeInfinity ScoreBoundType = 1
	// ScoreBoundPositiveInfinity 正无穷大 +inf
	ScoreBoundPositiveInfinity ScoreBoundType = 2
)

// SortedSetScoreBound 表示分数范围边界
type SortedSetScoreBound struct {
	Score   float64        // 分数值（仅当 Type == ScoreBoundValue 时有效）
	Exclude bool           // true 表示开区间（不包含该值），false 表示闭区间（包含该值）
	Type    ScoreBoundType // 边界类型
}

// String 将 SortedSetScoreBound 转换为 Redis 分数范围字符串
func (b SortedSetScoreBound) String() string {
	switch b.Type {
	case ScoreBoundNegativeInfinity:
		return "-inf"
	case ScoreBoundPositiveInfinity:
		return "+inf"
	}
	if b.Exclude {
		return fmt.Sprintf("(%v", b.Score)
	}
	return fmt.Sprintf("%v", b.Score)
}

// SortedSetZRangeByScore 使用 BYSCORE 模式查询有序集合，固定带 WITHSCORES
// min/max 为分数范围边界
// offset 跳过多少
// count 返回多少
func SortedSetZRangeByScore(ctx cd.AwaitableContext, index string, tableName string,
	dispatcher *cd.RedisMessageDispatcher, instance *redis.ClusterClient,
	min SortedSetScoreBound, max SortedSetScoreBound, offset int64, count int64, rev bool,
) (members []SortedSetRangeMember, retResult cd.RpcResult) {
	awaitOption := dispatcher.CreateDispatcherAwaitOptions()
	currentAction := ctx.GetAction()
	if lu.IsNil(currentAction) {
		ctx.LogError("not in context action")
		retResult = cd.CreateRpcResultError(fmt.Errorf("action not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	if currentAction.GetRpcContext() == nil || lu.IsNil(currentAction.GetRpcContext().GetContext()) {
		ctx.LogError("not found context")
		retResult = cd.CreateRpcResultError(fmt.Errorf("context not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}

	type innerPrivateData struct {
		Members []SortedSetRangeMember
	}

	pushActionFunc := func() cd.RpcResult {
		err := ctx.GetApp().PushAction(func(app_action *libatapp.AppActionData) error {
			ctx.GetApp().GetLogger(2).LogDebug("SortedSetZRangeByScore Send", "TableName", tableName, "Seq", awaitOption.Sequence, "index", index, "min", min.String(), "max", max.String(), "offset", offset, "count", count, "rev", rev)

			args := redis.ZRangeArgs{
				Key:     index,
				Start:   min.String(),
				Stop:    max.String(),
				ByScore: true,
				Rev:     rev,
				Offset:  offset,
				Count:   count,
			}
			cmdResult, redisError := instance.ZRangeArgsWithScores(ctx.GetContext(), args).Result()
			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}
			if redisError != nil {
				ctx.GetApp().GetLogger(2).LogError("SortedSetZRangeByScore Recv Error", "TableName", tableName, "Seq", awaitOption.Sequence, "redisError", redisError)
				resumeData.Result = cd.CreateRpcResultError(redisError, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
				resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("zrange by score failed resume error", "TableName", tableName, "err", resumeError)
					return resumeError
				}
				return redisError
			}
			resultMembers := make([]SortedSetRangeMember, 0, len(cmdResult))
			for _, z := range cmdResult {
				resultMembers = append(resultMembers, SortedSetRangeMember{
					Member: fmt.Sprintf("%v", z.Member),
					Score:  z.Score,
				})
			}
			ctx.GetApp().GetLogger(2).LogDebug("SortedSetZRangeByScore Recv Success", "TableName", tableName, "Seq", awaitOption.Sequence, "count", len(resultMembers))
			resumeData.PrivateData = &innerPrivateData{Members: resultMembers}
			resumeData.Result = cd.CreateRpcResultOk()
			resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
			if resumeError != nil {
				ctx.LogError("zrange by score failed resume error", "TableName", tableName, "err", resumeError)
				return resumeError
			}
			return nil
		}, nil, nil)
		if err != nil {
			return cd.CreateRpcResultError(err, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		}
		return cd.CreateRpcResultOk()
	}
	var resumeData *cd.DispatcherResumeData
	resumeData, retResult = cd.YieldTaskAction(ctx, currentAction, awaitOption, pushActionFunc, nil)
	if retResult.IsError() {
		return
	}
	if resumeData.Result.IsError() {
		retResult = resumeData.Result
		return
	}
	privateData, ok := resumeData.PrivateData.(*innerPrivateData)
	if !ok {
		ctx.LogError("zrange by score PrivateData failed not innerPrivateData")
		retResult = cd.CreateRpcResultError(fmt.Errorf("private data not innerPrivateData"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	members = privateData.Members
	return
}

// ============================================================================
// ZRank 相关结构与接口
// ============================================================================

// SortedSetZRankResult ZRank 返回结果（带 WITHSCORES）
type SortedSetZRankResult struct {
	Rank  int64   // 排名（0-based）
	Score float64 // 分数
}

// SortedSetZRank 获取成员的排名和分数（ZRANK WITHSCORES）
// rev=true 时使用 ZREVRANK
func SortedSetZRank(ctx cd.AwaitableContext, index string, tableName string,
	dispatcher *cd.RedisMessageDispatcher, instance *redis.ClusterClient,
	member string, rev bool,
) (result SortedSetZRankResult, found bool, retResult cd.RpcResult) {
	awaitOption := dispatcher.CreateDispatcherAwaitOptions()
	currentAction := ctx.GetAction()
	if lu.IsNil(currentAction) {
		ctx.LogError("not in context action")
		retResult = cd.CreateRpcResultError(fmt.Errorf("action not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	if currentAction.GetRpcContext() == nil || lu.IsNil(currentAction.GetRpcContext().GetContext()) {
		ctx.LogError("not found context")
		retResult = cd.CreateRpcResultError(fmt.Errorf("context not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}

	type innerPrivateData struct {
		Rank  int64
		Score float64
		Found bool
	}

	pushActionFunc := func() cd.RpcResult {
		err := ctx.GetApp().PushAction(func(app_action *libatapp.AppActionData) error {
			ctx.GetApp().GetLogger(2).LogDebug("SortedSetZRank Send", "TableName", tableName, "Seq", awaitOption.Sequence, "index", index, "member", member, "rev", rev)

			var rankResult *redis.RankWithScoreCmd
			if rev {
				rankResult = instance.ZRevRankWithScore(ctx.GetContext(), index, member)
			} else {
				rankResult = instance.ZRankWithScore(ctx.GetContext(), index, member)
			}

			resumeData := &cd.DispatcherResumeData{
				Message: &cd.DispatcherRawMessage{
					Type: awaitOption.Type,
				},
				Sequence:    awaitOption.Sequence,
				PrivateData: nil,
			}

			rankWithScore, redisError := rankResult.Result()
			if redisError != nil {
				if redisError == redis.Nil {
					// 成员不存在，不算错误
					ctx.GetApp().GetLogger(2).LogDebug("SortedSetZRank Member Not Found", "TableName", tableName, "Seq", awaitOption.Sequence, "member", member)
					resumeData.PrivateData = &innerPrivateData{Found: false}
					resumeData.Result = cd.CreateRpcResultOk()
					resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
					if resumeError != nil {
						ctx.LogError("zrank failed resume error", "TableName", tableName, "err", resumeError)
						return resumeError
					}
					return nil
				}
				ctx.GetApp().GetLogger(2).LogError("SortedSetZRank Recv Error", "TableName", tableName, "Seq", awaitOption.Sequence, "redisError", redisError)
				resumeData.Result = cd.CreateRpcResultError(redisError, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
				resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
				if resumeError != nil {
					ctx.LogError("zrank failed resume error", "TableName", tableName, "err", resumeError)
					return resumeError
				}
				return redisError
			}
			ctx.GetApp().GetLogger(2).LogDebug("SortedSetZRank Recv Success", "TableName", tableName, "Seq", awaitOption.Sequence, "rank", rankWithScore.Rank, "score", rankWithScore.Score)
			resumeData.PrivateData = &innerPrivateData{
				Rank:  rankWithScore.Rank,
				Score: rankWithScore.Score,
				Found: true,
			}
			resumeData.Result = cd.CreateRpcResultOk()
			resumeError := cd.ResumeTaskAction(ctx, currentAction, resumeData)
			if resumeError != nil {
				ctx.LogError("zrank failed resume error", "TableName", tableName, "err", resumeError)
				return resumeError
			}
			return nil
		}, nil, nil)
		if err != nil {
			return cd.CreateRpcResultError(err, public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		}
		return cd.CreateRpcResultOk()
	}
	var resumeData *cd.DispatcherResumeData
	resumeData, retResult = cd.YieldTaskAction(ctx, currentAction, awaitOption, pushActionFunc, nil)
	if retResult.IsError() {
		return
	}
	if resumeData.Result.IsError() {
		retResult = resumeData.Result
		return
	}
	privateData, ok := resumeData.PrivateData.(*innerPrivateData)
	if !ok {
		ctx.LogError("zrank PrivateData failed not innerPrivateData")
		retResult = cd.CreateRpcResultError(fmt.Errorf("private data not innerPrivateData"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
		return
	}
	result.Rank = privateData.Rank
	result.Score = privateData.Score
	found = privateData.Found
	return
}

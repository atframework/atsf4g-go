package lobbysvr_logic_user_auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
	"google.golang.org/protobuf/types/known/timestamppb"

	config "github.com/atframework/atsf4g-go/component/config"
	cd "github.com/atframework/atsf4g-go/component/dispatcher"

	private_protocol_config "github.com/atframework/atsf4g-go/component/protocol/private/config/protocol/config"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/private/pbdesc/protocol/pbdesc"
)

type Argon2Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

var defaultParams = Argon2Params{
	Memory:      64 * 1024,
	Iterations:  3,
	Parallelism: 1,
	SaltLength:  16,
	KeyLength:   32,
}

func GeneratePasswordHash(ctx cd.RpcContext, password string) (string, error) {
	lobbySvrCfg := config.GetServerConfig[*private_protocol_config.Readonly_LobbyServerCfg](config.GetConfigManager().GetCurrentConfigGroup())
	if lobbySvrCfg == nil {
		ctx.LogError("Lobby server config is nil")
		return "", fmt.Errorf("lobby server config is nil")
	}

	p := &Argon2Params{
		Memory:      lobbySvrCfg.GetAuth().GetArgon2().GetMemory(),
		Iterations:  lobbySvrCfg.GetAuth().GetArgon2().GetIterations(),
		Parallelism: uint8(lobbySvrCfg.GetAuth().GetArgon2().GetParallelism()),
		SaltLength:  lobbySvrCfg.GetAuth().GetArgon2().GetSaltLength(),
		KeyLength:   lobbySvrCfg.GetAuth().GetArgon2().GetKeyLength(),
	}

	salt := make([]byte, p.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)
	b64 := base64.RawStdEncoding
	encodedSalt := b64.EncodeToString(salt)
	encodedHash := b64.EncodeToString(hash)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		p.Memory,
		p.Iterations,
		p.Parallelism,
		encodedSalt,
		encodedHash,
	)

	return encoded, nil
}

func VerifyPassword(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, errors.New("invalid hash format")
	}

	if parts[1] != "argon2id" {
		return false, errors.New("unsupported algorithm")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, err
	}
	if version != argon2.Version {
		return false, errors.New("incompatible argon2 version")
	}

	var p Argon2Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Iterations, &p.Parallelism); err != nil {
		return false, err
	}

	b64 := base64.RawStdEncoding

	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	hash, err := b64.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	// p.SaltLength = uint32(len(salt))
	p.KeyLength = uint32(len(hash))

	otherHash := argon2.IDKey([]byte(password), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)

	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return true, nil
	}
	return false, nil
}

func HasPasswordWeakToken(ctx cd.RpcContext, weakToken string, data *private_protocol_pbdesc.DatabaseAuthWeakData) bool {
	if weakToken == "" {
		return false
	}
	if data == nil {
		return false
	}

	now := ctx.GetNow()
	for _, token := range data.GetTokenList() {
		if token.GetPassword().GetWeakToken() == weakToken {
			if token.GetPassword().GetExpiredTimepoint().AsTime().After(now) {
				return true
			}
		}
	}

	return false
}

func AddWeakToken(ctx cd.RpcContext, weakToken string, expiredTime time.Time, data *private_protocol_pbdesc.DatabaseAuthWeakData) {
	if weakToken == "" {
		return
	}

	maxCount := 10
	lobbySvrCfg := config.GetServerConfig[*private_protocol_config.Readonly_LobbyServerCfg](config.GetConfigManager().GetCurrentConfigGroup())
	if lobbySvrCfg == nil {
		ctx.LogError("Lobby server config is nil")
	}
	if lobbySvrCfg.GetAuth().GetWeakTokenMaxCount() > 0 {
		maxCount = int(lobbySvrCfg.GetAuth().GetWeakTokenMaxCount())
	}

	if data == nil {
		ctx.LogError("DatabaseAuthWeakData is nil")
		return
	}

	inserted := false
	for _, token := range data.MutableTokenList() {
		if token.GetPassword().GetWeakToken() == weakToken {
			token.MutablePassword().ExpiredTimepoint = timestamppb.New(expiredTime)
			inserted = true
			break
		}
	}

	if !inserted {
		newToken := data.AddTokenList()
		newToken.MutablePassword().WeakToken = weakToken
		newToken.MutablePassword().ExpiredTimepoint = timestamppb.New(expiredTime)
	}

	if len(data.GetTokenList()) > maxCount {
		// 按过期时间排序，删除过期时间最远的
		tokens := data.GetTokenList()
		sort.Slice(tokens, func(i, j int) bool {
			return tokens[i].GetPassword().GetExpiredTimepoint().AsTime().Before(tokens[j].GetPassword().GetExpiredTimepoint().AsTime())
		})
		for i := maxCount; i < len(tokens); i++ {
			ctx.LogDebug("token %s is removed", tokens[i].GetPassword().GetWeakToken())
		}
		data.TokenList = tokens[:maxCount]
	}
}

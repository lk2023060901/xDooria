package manager

import (
	"context"
	"fmt"

	"github.com/lk2023060901/xdooria/component/auth"
	"github.com/lk2023060901/xdooria/app/login/internal/gameconfig"
	pb "github.com/lk2023060901/xdooria-proto-common"
	"google.golang.org/protobuf/proto"
)

type LocalAuthenticator struct{}

func NewLocalAuthenticator() *LocalAuthenticator {
	return &LocalAuthenticator{}
}

func (a *LocalAuthenticator) Type() pb.LoginType {
	return pb.LoginType_LOGIN_TYPE_LOCAL
}

func (a *LocalAuthenticator) Authenticate(ctx context.Context, cred []byte) (*auth.Identity, error) {
	// 1. 解析凭证
	var localCred pb.AccountPassword
	if err := proto.Unmarshal(cred, &localCred); err != nil {
		return nil, fmt.Errorf("failed to unmarshal local credentials: %w", err)
	}

	// 2. 在全局配置表中查找账号
	var foundAccount *gameconfig.Account
	for _, acc := range gameconfig.T.TbAccount.GetDataList() {
		if acc.UserName == localCred.Username {
			foundAccount = acc
			break
		}
	}

	if foundAccount == nil {
		return nil, fmt.Errorf("account not found: %s", localCred.Username)
	}

	// 3. 校验密码 (实际开发中应使用加密/哈希)
	if foundAccount.Password != localCred.Password {
		return nil, fmt.Errorf("invalid password")
	}

	// 4. 返回身份信息
	return &auth.Identity{
		UID:      fmt.Sprintf("%d", foundAccount.Id),
		Nickname: foundAccount.UserName,
	}, nil
}

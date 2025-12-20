package user

import "github.com/lk2023060901/xdooria/pkg/grpc/interceptor"

// Validate 实现 Validator 接口（参数校验）
func (req *CreateUserRequest) Validate() error {
	errs := interceptor.NewMultiValidationError()

	if req.Username == "" {
		errs.Add(interceptor.NewValidationError("username", "cannot be empty"))
	}
	if len(req.Username) < 3 {
		errs.Add(interceptor.NewValidationError("username", "must be at least 3 characters"))
	}
	if req.Email == "" {
		errs.Add(interceptor.NewValidationError("email", "cannot be empty"))
	}
	if req.Age < 0 || req.Age > 150 {
		errs.Add(interceptor.NewValidationError("age", "must be between 0 and 150"))
	}

	return errs.ToError()
}

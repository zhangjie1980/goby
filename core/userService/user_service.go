package userService

import (
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Unknwon/com"

	. "github.com/MessageDream/goby/core"
	"github.com/MessageDream/goby/model"
	"github.com/MessageDream/goby/module/infrastructure"
	"github.com/MessageDream/goby/module/setting"
)

var (
	reservedUsernames    = []string{"assets", "css", "img", "js", "less", "plugins", "debug", "raw", "install", "api", "avatar", "user", "template", "admin", "new", ".", ".."}
	reservedUserPatterns = []string{"*.keys"}
)

func isUsableName(names, patterns []string, name string) error {
	name = strings.TrimSpace(strings.ToLower(name))
	if utf8.RuneCountInString(name) == 0 {
		return ErrNameEmpty
	}

	for i := range names {
		if name == names[i] {
			return ErrNameReserved
		}
	}

	for _, pat := range patterns {
		if pat[0] == '*' && strings.HasSuffix(name, pat[1:]) ||
			(pat[len(pat)-1] == '*' && strings.HasPrefix(name, pat[:len(pat)-1])) {
			return WrapIntentError(fmt.Errorf("User name like %s can't be allowned.", pat), INTENT_ERROR_CODE_USER_NAME_NOT_ALLOWNED)
		}
	}

	return nil
}

func verifyActiveCode(code string) (*model.User, error) {
	minutes := setting.Service.ActiveCodeLives
	limitLength := infrastructure.TimeLimitCodeLength

	if len(code) <= limitLength {
		return nil, ErrUserActivateTimeLimitCodeLength
	}

	hexStr := code[limitLength:]

	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		LowerName: string(b),
	}

	exist, er := user.Get()

	if er != nil {
		return nil, er
	}
	if !exist {
		return nil, ErrUserNotExist
	}

	data := com.ToStr(user.ID) + user.Email + user.LowerName + user.Password + user.Rands
	prefix := code[:limitLength]

	if infrastructure.VerifyTimeLimitCode(data, minutes, prefix) {
		return user, nil
	}
	return nil, ErrUserActivateVerifyFailed
}

func Active(code string) (*model.User, error) {
	user, err := verifyActiveCode(code)
	if err != nil {
		return nil, err
	}

	if user.IsActive {
		return nil, ErrUserAlreadyActivated
	}

	user.IsActive = true
	user.GenerateRands()
	if err := user.Update(nil, "is_active", "rands"); err != nil {
		return nil, err
	}

	return user, nil
}

func Create(uname, pwd, email string, isActive, isAdmin bool) (*model.User, error) {

	if err := isUsableName(reservedUsernames, reservedUserPatterns, uname); err != nil {
		return nil, err
	}

	user := &model.User{
		UserName:  uname,
		LowerName: strings.ToLower(uname),
		Password:  pwd,
		Email:     email,
		IsActive:  isActive,
		IsAdmin:   isAdmin,
	}
	if exist, err := user.Exist(); err != nil || exist {
		if exist {
			return nil, ErrUserNameAlreadyExist
		}
		return nil, err
	}
	if can, err := user.EmailUsable(); err != nil || !can {
		if !can {
			return nil, ErrEmailAlreadyExist
		}
		return nil, err
	}

	return user, user.Create(nil)

}

func GetByID(uid uint64) (*model.User, error) {
	user := &model.User{ID: uid}

	if exist, err := user.Get(); err != nil || !exist {
		if !exist {
			return nil, ErrUserNotExist
		}
		return nil, err
	}

	return user, nil
}

func GetByRands(rands string) (*model.User, error) {
	user := &model.User{Rands: rands}

	if exist, err := user.Get(); err != nil || !exist {
		if !exist {
			return nil, ErrUserNotExist
		}
		return nil, err
	}
	return user, nil
}

func GetByUserName(uname string) (*model.User, error) {
	user := &model.User{LowerName: strings.ToLower(uname)}

	if exist, err := user.Get(); err != nil || !exist {
		if !exist {
			return nil, ErrUserNotExist
		}
		return nil, err
	}
	return user, nil
}

func Signin(emailOrName, pwd string) (*model.User, error) {
	var user *model.User
	if strings.Contains(emailOrName, "@") {
		if !infrastructure.VerifyEmail(emailOrName) || len(pwd) <= 0 {
			return nil, ErrUserNameOrPasswordInvalide
		}
		user = &model.User{Email: strings.ToLower(emailOrName)}
	} else {
		user = &model.User{LowerName: strings.ToLower(emailOrName)}
	}

	if len(pwd) <= 0 {
		return nil, ErrUserNameOrPasswordInvalide
	}

	if exist, err := user.Get(); err != nil || !exist {
		if !exist {
			return nil, ErrUserNotExist
		}
		return nil, err
	}

	if !user.ValidatePassword(pwd) {
		return nil, ErrUserNameOrPasswordInvalide
	}
	return user, nil
}

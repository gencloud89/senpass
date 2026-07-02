package services

import (
	"crypto/rand"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"os"

	"nyanpass-backend/internal/database"
	"nyanpass-backend/internal/models"
)

type AuthService struct{}

// Login xác thực user và tạo token. remember=true → token hết hạn sau 7 ngày
func (s *AuthService) Login(username, password string, remember bool) (string, error) {
	var user models.User
	if err := database.DB.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("用户名或密码错误")
		}
		return "", err
	}

	if user.Banned {
		return "", errors.New("账户已被禁用")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errors.New("用户名或密码错误")
	}

	// Tạo token và lưu vào user_logins
	token := generateToken()
	login := models.UserLogin{
		UID:  user.ID,
		Token: token,
	}
	if remember {
		login.TokenExpire = currentUnix() + 7*24*3600 // 7 ngày
	}
	if err := database.DB.Create(&login).Error; err != nil {
		return "", err
	}

	return token, nil
}

// HashPassword băm mật khẩu với bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// SeedDefaultUser tạo tài khoản admin từ biến môi trường nếu chưa tồn tại
// ADMIN_USER và ADMIN_PASS được set bởi install script
func SeedDefaultUser() error {
	adminUser := os.Getenv("ADMIN_USER")
	adminPass := os.Getenv("ADMIN_PASS")
	if adminUser == "" || adminPass == "" {
		return nil // Không có env → bỏ qua, admin phải tạo thủ công
	}

	var count int64
	database.DB.Model(&models.User{}).Where("username = ?", adminUser).Count(&count)
	if count > 0 {
		return nil
	}

	hash, err := HashPassword(adminPass)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user := models.User{
		Username:      adminUser,
		Password:      hash,
		Admin:         true,
		MaxRules:      999999999,
		TrafficEnable: 1073741822926258200,
		Expire:        253392455349,
		GroupID:       1,
	}
	return database.DB.Create(&user).Error
}

// isValidToken kiểm tra token trong user_logins (dùng cho middleware)
func IsValidToken(token string) (uint64, error) {
	var login models.UserLogin
	if err := database.DB.Where("token = ?", token).First(&login).Error; err != nil {
		return 0, err
	}
	return login.UID, nil
}

func currentUnix() int64 {
	return database.DB.NowFunc().Unix()
}

func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

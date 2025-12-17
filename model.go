package main

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

// User 模型定义
type User struct {
	ID         uint64         `gorm:"column:id;type:bigint unsigned;primaryKey;autoIncrement" json:"id"`
	EmployeeNo string         `gorm:"column:employee_no;type:varchar(32);not null;unique" json:"employee_no"`
	Name       string         `gorm:"column:username;type:varchar(32);not null" json:"username"`
	Password   string         `gorm:"column:password;type:varchar(255);not null" json:"-"`
	Gender     uint8          `gorm:"column:gender;type:tinyint unsigned;default:0" json:"gender,omitempty"`
	Age        uint8          `gorm:"column:age;type:tinyint unsigned;null" json:"age,omitempty"`
	Email      string         `gorm:"column:email;type:varchar(64);unique;null" json:"email,omitempty"`
	Mobile     string         `gorm:"column:mobile;type:varchar(11);not null;unique" json:"mobile"`
	Department string         `gorm:"column:department;type:varchar(64);not null" json:"department"`
	Position   string         `gorm:"column:position;type:varchar(64);not null" json:"position"`
	Rank       string         `gorm:"column:rank;type:varchar(32);default:''" json:"rank,omitempty"`
	WorkStatus uint8          `gorm:"column:work_status;type:tinyint unsigned;default:1" json:"work_status"`
	Salary     float64        `gorm:"column:salary;type:decimal(12,2);not null" json:"salary,omitempty"`
	CreateTime time.Time      `gorm:"column:create_time;type:datetime(3);default:current_timestamp(3)" json:"create_time"`
	UpdateTime time.Time      `gorm:"column:update_time;type:datetime(3);autoUpdateTime" json:"update_time,omitempty"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;type:datetime(3);null" json:"-"`
}

func (u *User) TableName() string {
	return "users"
}

type LoginReq struct {
	LoginID  string `json:"login_id" binding:"required"` // 工号/手机号/邮箱
	Password string `json:"password" binding:"required"`
}

type RegisterReq struct {
	EmployeeNo string  `json:"employee_no" binding:"required"`
	Name       string  `json:"name" binding:"required"`
	Password   string  `json:"password" binding:"required"`
	Gender     uint8   `json:"gender,omitempty"`
	Age        uint8   `json:"age,omitempty"`
	Email      string  `json:"email,omitempty"`
	Mobile     string  `json:"mobile" binding:"required"`
	Department string  `json:"department" binding:"required"`
	Position   string  `json:"position" binding:"required"`
	Rank       string  `json:"rank,omitempty"`
	WorkStatus uint8   `json:"work_status,omitempty"` // 默认1-在职
	Salary     float64 `json:"salary" binding:"required"`
}

type UserListReq struct {
	Page       int    `json:"page" default:"1"`       // 页码
	PageSize   int    `json:"page_size" default:"10"` // 每页条数
	Department string `json:"department,omitempty"`   // 按部门筛选
	WorkStatus uint8  `json:"work_status,omitempty"`  // 按在职状态筛选
}

type UserListResp struct {
	Total int64  `json:"total"` // 总条数
	List  []User `json:"list"`  // 员工列表
}

// 修复：Claims 的 UserID 改为 uint64，与 User.ID 类型一致
type Claims struct {
	UserID     uint64 `json:"user_id"`
	EmployeeNo string `json:"employee_no"`
	jwt.RegisteredClaims
}

func EncryptPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func RegisterEmployee(req *RegisterReq) error {
	var exist User
	if err := DB.Where("employee_no = ?", req.EmployeeNo).First(&exist).Error; err == nil {
		return errors.New("员工工号已存在")
	}
	if err := DB.Where("mobile = ?", req.Mobile).First(&exist).Error; err == nil {
		return errors.New("手机号码已存在")
	}
	if req.Email != "" {
		if err := DB.Where("email = ?", req.Email).First(&exist).Error; err == nil {
			return errors.New("邮箱已存在")
		}
	}

	hashPwd, err := EncryptPassword(req.Password)
	if err != nil {
		return errors.New("密码加密失败：" + err.Error())
	}

	user := &User{
		EmployeeNo: req.EmployeeNo,
		Name:       req.Name,
		Password:   hashPwd,
		Gender:     req.Gender,
		Age:        req.Age,
		Email:      req.Email,
		Mobile:     req.Mobile,
		Department: req.Department,
		Position:   req.Position,
		Rank:       req.Rank,
		WorkStatus: req.WorkStatus,
		Salary:     req.Salary,
	}

	if user.WorkStatus == 0 {
		user.WorkStatus = 1
	}
	return DB.Create(user).Error
}

func LoginEmployee(req *LoginReq) (string, error) {
	var user User
	err := DB.Where("employee_no = ? OR mobile = ? OR email = ?", req.LoginID, req.LoginID, req.LoginID).First(&user).Error
	if err != nil {
		return "", errors.New("用户不存在")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return "", errors.New("密码错误")
	}

	if user.WorkStatus != 1 && user.WorkStatus != 3 {
		return "", errors.New("账号已离职，无法登录")
	}

	claims := Claims{
		UserID:     user.ID, // 类型匹配，不再报错
		EmployeeNo: user.EmployeeNo,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "employee-system",
		},
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := jwtToken.SignedString([]byte("mvc-secret-123"))
	if err != nil {
		return "", errors.New("生成token失败：" + err.Error())
	}

	return tokenStr, nil
}

func GetUserList(req *UserListReq) (*UserListResp, error) {
	db := DB.Model(&User{}).Where("deleted_at IS NULL") // 排除软删除数据
	if req.Department != "" {
		db = db.Where("department = ?", req.Department)
	}
	if req.WorkStatus > 0 {
		db = db.Where("work_status = ?", req.WorkStatus)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, errors.New("统计总数失败：" + err.Error())
	}
	//分页
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 10
	}
	offset := (req.Page - 1) * req.PageSize
	// 查询列表数据
	var list []User
	if err := db.Order("create_time DESC").Offset(offset).Limit(req.PageSize).Find(&list).Error; err != nil {
		return nil, errors.New("查询列表失败：" + err.Error())
	}
	return &UserListResp{
		Total: total,
		List:  list,
	}, nil
}

// 修复：删除显性建表语句，仅保留 AutoMigrate 自动建表
func InitDB() {
	dsn := "root:888888@tcp(127.0.0.1:3306)/temp?charset=utf8mb4&parseTime=True&loc=Local"
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("数据库连接失败：" + err.Error())
	}
	// 仅保留 AutoMigrate 自动同步表结构，无需显性建表SQL
	if err := DB.AutoMigrate(&User{}); err != nil {
		panic("表结构迁移失败：" + err.Error())
	}
}

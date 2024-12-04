package user

type User struct {
	ID        int64
	Username  string `binding:"required"`
	Password  string `binding:"required"`
	CreatedAt string
}

func NewUserModel() *User {
	return &User{}
}

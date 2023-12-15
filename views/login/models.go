package login

type RegistrationForm struct {
	Email                string `form:"email"`
	Password             string `form:"password"`
	PasswordConfirmation string `form:"password_confirmation"`
}

type LoginForm struct {
	Email    string `form:"email"`
	Password string `form:"password"`
}

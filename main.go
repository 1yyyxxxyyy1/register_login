package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	InitDB()
	r := gin.Default()
	r.SetTrustedProxies([]string{"127.0.0.1"})
	userCtrl := &UserController{}
	r.POST("/register", userCtrl.Register)
	r.POST("/login", userCtrl.Login)
	r.GET("/user/list", userCtrl.List)

	r.Run(":8080")
}

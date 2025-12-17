package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserController struct{}

// 注册请求的处理,绑定参数，以及调用方法
func (uc *UserController) Register(c *gin.Context) {
	var req RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		fmt.Println(err)
		return
	}

	if err := RegisterEmployee(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "注册成功"})
}

func (uc *UserController) Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	tokenStr, err := LoginEmployee(&req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "登陆成功", "token": tokenStr})
}

func (uc *UserController) List(c *gin.Context) {
	var req UserListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误：" + err.Error()})
		return
	}
	listResp, err := GetUserList(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询列表失败：" + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "查询成功",
		"data": listResp,
	})
}

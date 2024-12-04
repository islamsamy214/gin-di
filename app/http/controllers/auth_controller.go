package controllers

type AuthController struct{}

// func (*AuthController) Login(ctx *gin.Context) {
// 	user := models.User{}
// 	err := ctx.ShouldBindJSON(&user)

// 	if err != nil {
// 		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	loginUser, err := (&user).Login()
// 	if err != nil {
// 		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	token, err := helpers.GenerateToken(loginUser.ID)

// 	if err != nil {
// 		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	ctx.JSON(http.StatusOK, gin.H{
// 		"message": "Success",
// 		"user":    loginUser,
// 		"token":   token,
// 	})
// }

// func (*AuthController) Signup(ctx *gin.Context) {
// 	user := models.User{}
// 	err := ctx.ShouldBindJSON(&user)

// 	if err != nil {
// 		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	createdUser, err := (&user).Create()

// 	if err != nil {
// 		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	ctx.JSON(http.StatusOK, gin.H{
// 		"message": "Success",
// 		"user":    createdUser,
// 	})
// }

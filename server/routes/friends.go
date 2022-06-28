/* The /friends group contains all the routes related to adding/removing/requesting friends */
package routes

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"schej.it/server/db"
	"schej.it/server/errs"
	"schej.it/server/logger"
	"schej.it/server/middleware"
	"schej.it/server/models"
	"schej.it/server/responses"
)

func InitFriends(router *gin.Engine) {
	friendsRouter := router.Group("/friends")
	friendsRouter.Use(middleware.AuthRequired())

	friendsRouter.GET("")
	friendsRouter.DELETE("/:id")
	friendsRouter.GET("/requests")
	friendsRouter.POST("/requests", createFriendRequest)
	friendsRouter.POST("/requests/:id/accept", acceptFriendRequest)
	friendsRouter.POST("/requests/:id/reject", rejectFriendRequest)
	friendsRouter.DELETE("/requests/:id")
}

// @Summary Creates a new friend request
// @Tags friends
// @Accept json
// @Produce json
// @Param from body primitive.ObjectId true "The sender of the friend request"
// @Param to body primitive.ObjectId true "The recipient of the friend request"
// @Success 201 {object} models.FriendRequest
// @Router /requests [post]
func createFriendRequest(c *gin.Context) {
	payload := struct {
		From primitive.ObjectID `json:"from" binding:"required"`
		To   primitive.ObjectID `json:"to" binding:"required"`
	}{}
	if err := c.Bind(&payload); err != nil {
		return
	}

	friendRequest := models.FriendRequest{
		From:      payload.From,
		To:        payload.To,
		CreatedAt: primitive.NewDateTimeFromTime(time.Now()),
	}

	// Insert friend request
	result, err := db.FriendRequestsCollection.InsertOne(context.Background(), friendRequest)
	if err != nil {
		logger.StdErr.Panicln(err)
	}

	insertedId := result.InsertedID.(primitive.ObjectID)
	friendRequest.Id = insertedId

	// Populate the ToUser field
	friendRequest.ToUser = db.GetUserById(friendRequest.To.Hex()).GetProfile()
}

// @Summary Accepts an existing friend request
// @Tags friends
// @Accept json
// @Produce json
// @Param id path string true "ID of the friend request"
// @Success 200
// @Router /requests/:id/accept [post]
func acceptFriendRequest(c *gin.Context) {
	friendRequestId := c.Param("id")
	friendRequest := db.GetFriendRequestById(friendRequestId)
	if friendRequest == nil {
		c.JSON(http.StatusNotFound, responses.Error{Error: errs.FriendRequestNotFound})
		return
	}

	// Check if the "To" user id matches the current user's id
	userInterface, _ := c.Get("authUser")
	user := userInterface.(*models.User)
	if user.Id != friendRequest.To {
		c.JSON(http.StatusForbidden, gin.H{})
		return
	}

	// user1 := db.GetUserById(friendRequest.From.Hex())
	// if user1 == nil {
	// 	logger.StdErr.Panicf("User(id=%s) does not exist!\n", friendRequest.From.Hex())
	// }

	// user2 := db.GetUserById(friendRequest.To.Hex())
	// if user2 == nil {
	// 	logger.StdErr.Panicf("User(id=%s) does not exist!\n", friendRequest.To.Hex())
	// }

	// Update friend arrays of both From and To user
	db.UsersCollection.UpdateOne(context.Background(),
		bson.M{"$and": bson.A{
			bson.M{"_id": friendRequest.To},
			bson.M{"friendIds": bson.M{"$ne": bson.A{friendRequest.From}}},
		}},
		bson.M{"$push": bson.M{"friendIds": friendRequest.From}},
	)

	db.UsersCollection.UpdateOne(context.Background(),
		bson.M{"$and": bson.A{
			bson.M{"_id": friendRequest.From},
			bson.M{"friendIds": bson.M{"$ne": bson.A{friendRequest.To}}},
		}},
		bson.M{"$push": bson.M{"friendIds": friendRequest.To}},
	)

	// Delete friend request

	c.JSON(http.StatusOK, gin.H{})
}

// @Summaryejects an existing friend request
// @Tags friends
// @Accept json
// @Produce json
// @Param id path string true "ID of the friend request"
// @Success 200
// @Router /requests/:id/reject [post]
func rejectFriendRequest(c *gin.Context) {
	friendRequestId := c.Param("id")
	friendRequest := db.GetFriendRequestById(friendRequestId)
	if friendRequest == nil {
		c.JSON(http.StatusNotFound, responses.Error{Error: errs.FriendRequestNotFound})
		return
	}

	// Check if the "To" user id matches the current user's id
	userInterface, _ := c.Get("authUser")
	user := userInterface.(*models.User)
	if user.Id != friendRequest.To {
		c.JSON(http.StatusForbidden, gin.H{})
		return
	}

	// Delete friend request

	c.JSON(http.StatusOK, gin.H{})
}

package controllers

import (
	"net/http"
	"os"
	"strconv"
	"tusk/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TaskController struct {
	DB *gorm.DB
}

func (t *TaskController) CreateTask(c *gin.Context) {
	task := models.Task{}
	errBindJson := c.ShouldBindJSON(&task)
	if errBindJson != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errBindJson.Error()})
		return
	}

	errDb := t.DB.Create(&task).Error
	if errDb != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errDb.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (t *TaskController) Delete(c *gin.Context) {
	id := c.Param("id")
	task := models.Task{}

	if err := t.DB.First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data Not Found"})
		return
	}

	errDb := t.DB.Delete(&models.Task{}, id).Error
	if errDb != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errDb.Error()})
		return
	}

	if task.Attachment != "" {
		os.Remove("attachments/" + task.Attachment)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Delete Task Success"})
}

func (t *TaskController) Submit(c *gin.Context) {
	task := models.Task{}
	id := c.Param("id")
	submitDate := c.PostForm("submitDate")

	file, errFile := c.FormFile("attachment")
	if errFile != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errFile.Error()})
		return
	}

	if err := t.DB.First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data Not Found"})
		return
	}

	//remove old attachment
	attachment := task.Attachment
	fileInfo, _ := os.Stat("attachements/" + attachment)
	if fileInfo != nil {
		os.Remove("attachments/" + attachment)
	}

	attachment = file.Filename
	errSave := c.SaveUploadedFile(file, "attachments/"+attachment)
	if errSave != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errSave.Error()})
		return
	}

	errDb := t.DB.Where("id=?", id).Updates(models.Task{
		Status:     "Review",
		SubmitDate: submitDate,
		Attachment: attachment,
	}).Error

	if errDb != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errDb.Error()})
		return
	}

	c.JSON(http.StatusOK, "Submit to Review")
}

func (t *TaskController) Reject(c *gin.Context) {
	task := models.Task{}
	id := c.Param("id")
	rejectedDate := c.PostForm("rejectedDate")
	reason := c.PostForm("reason")

	if err := t.DB.First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data Not Found"})
		return
	}

	errDb := t.DB.Where("id=?", id).Updates(models.Task{
		Status:       "Rejected",
		Reason:       reason,
		RejectedDate: rejectedDate,
	}).Error

	if errDb != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errDb.Error()})
		return
	}

	c.JSON(http.StatusOK, "Rejected")
}

func (t *TaskController) Fix(c *gin.Context) {

	id := c.Param("id")
	revision, errConv := strconv.Atoi(c.PostForm("revision"))
	if errConv != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errConv.Error()})
		return
	}

	if err := t.DB.First(&models.Task{}, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data Not Found"})
		return
	}

	errDb := t.DB.Where("id=?", id).Updates(models.Task{
		Status:   "Queue",
		Revision: int8(revision),
	}).Error

	if errDb != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errDb.Error()})
		return
	}

	c.JSON(http.StatusOK, "Fix to Queue")
}

func (t *TaskController) Approve(c *gin.Context) {

	id := c.Param("id")
	approvedDate := c.PostForm("approvedDate")

	if err := t.DB.First(&models.Task{}, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data Not Found"})
		return
	}

	errDb := t.DB.Where("id=?", id).Updates(models.Task{
		Status:      "Approved",
		ApproveDate: approvedDate,
	}).Error

	if errDb != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errDb.Error()})
		return
	}

	c.JSON(http.StatusOK, "Approved")
}

func (t *TaskController) FindTaskById(c *gin.Context) {
	task := models.Task{}
	id := c.Param("id")

	if err := t.DB.First(&models.Task{}, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data Not Found"})
		return
	}

	errDB := t.DB.Preload("User").Find(&task, id).Error
	if errDB != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errDB.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (t *TaskController) NeedToBeReview(c *gin.Context) {
	tasks := []models.Task{}

	errDB := t.DB.Preload("User").Where("status=?", "Review").Order("submit_date ASC").Limit(2).Find(&tasks).Error
	if errDB != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errDB.Error()})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

func (t *TaskController) ProgressTask(c *gin.Context) {
	tasks := []models.Task{}
	userId := c.Param("userId")

	errDB := t.DB.Where("(status !=? AND user_id=?) OR (revision !=? AND user_id=?)", "Queue", userId, 0, userId).Order("updated_at ASC").Limit(5).Find(&tasks).Error
	if errDB != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errDB.Error()})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

func (t *TaskController) Statistic(c *gin.Context) {
	userId := c.Param("userId")

	stat := []map[string]interface{}{}

	errDB := t.DB.Model(models.Task{}).Select("status,count(status) as total").Where(
		"user_id=?", userId).Group("status").Find(&stat).Error
	if errDB != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errDB.Error()})
		return
	}

	c.JSON(http.StatusOK, stat)
}

func (t *TaskController) FindByUserAndStatus(c *gin.Context) {
	tasks := []models.Task{}
	userId := c.Param("userId")
	status := c.Param("status")

	errDB := t.DB.Where("user_id=? AND status=?", userId, status).Find(&tasks).Error
	if errDB != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": errDB.Error()})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

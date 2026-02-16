package course_recommendation

import (
	"errors"
	"time"

	"github.com/W1ndys/easy-qfnu-api-go/common/notify"
	"github.com/W1ndys/easy-qfnu-api-go/internal/database"
	"github.com/W1ndys/easy-qfnu-api-go/model"
)

var (
	ErrNotFound = errors.New("推荐记录不存在")
)

// Query 根据关键词查询可见的课程推荐（匹配课程名称或教师姓名）
func Query(keyword string) ([]model.CourseRecommendationPublic, error) {
	db := database.GetCourseRecDB()
	if db == nil {
		return nil, errors.New("数据库连接失败")
	}

	pattern := "%" + keyword + "%"
	rows, err := db.Query(`
		SELECT course_name, teacher_name, recommendation_reason, recommender_nickname, recommendation_time, campus, recommendation_year
		FROM course_recommendations
		WHERE is_visible = 1 AND (course_name LIKE ? OR teacher_name LIKE ?)
		ORDER BY recommendation_time DESC
	`, pattern, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.CourseRecommendationPublic
	for rows.Next() {
		var r model.CourseRecommendationPublic
		if err := rows.Scan(&r.CourseName, &r.TeacherName, &r.RecommendationReason, &r.RecommenderNickname, &r.RecommendationTime, &r.Campus, &r.RecommendationYear); err != nil {
			continue
		}
		list = append(list, r)
	}

	if list == nil {
		return []model.CourseRecommendationPublic{}, nil
	}
	return list, nil
}

// Recommend 提交课程推荐
func Recommend(req model.CourseRecommendationRecommendRequest) (int64, error) {
	db := database.GetCourseRecDB()
	if db == nil {
		return 0, errors.New("数据库连接失败")
	}

	nickname := req.RecommenderNickname
	if nickname == "" {
		nickname = "匿名"
	}

	now := time.Now().Unix()
	_, err := db.Exec(`
		INSERT INTO course_recommendations (course_name, teacher_name, recommendation_reason, recommender_nickname, recommendation_time, is_visible, campus, recommendation_year)
		VALUES (?, ?, ?, ?, ?, 0, ?, ?)
	`, req.CourseName, req.TeacherName, req.RecommendationReason, nickname, now, req.Campus, req.RecommendationYear)
	if err != nil {
		return 0, err
	}

	// 发送飞书通知
	notify.NotifyNewRecommendation(req.CourseName, req.TeacherName, nickname, req.RecommendationReason)

	return now, nil
}

// Update 更新课程推荐信息（管理员用）
func Update(req model.CourseRecommendationUpdateRequest) error {
	db := database.GetCourseRecDB()
	if db == nil {
		return errors.New("数据库连接失败")
	}

	nickname := req.RecommenderNickname
	if nickname == "" {
		nickname = "匿名"
	}

	visibleInt := 0
	if req.IsVisible {
		visibleInt = 1
	}

	result, err := db.Exec(`
		UPDATE course_recommendations
		SET course_name = ?, teacher_name = ?, recommendation_reason = ?, recommender_nickname = ?, is_visible = ?, campus = ?, recommendation_year = ?
		WHERE id = ?
	`, req.CourseName, req.TeacherName, req.RecommendationReason, nickname, visibleInt, req.Campus, req.RecommendationYear, req.RecommendationID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// Review 审核课程推荐（设置是否可见）
func Review(recommendationID int64, isVisible bool) error {
	db := database.GetCourseRecDB()
	if db == nil {
		return errors.New("数据库连接失败")
	}

	visibleInt := 0
	if isVisible {
		visibleInt = 1
	}

	result, err := db.Exec(`
		UPDATE course_recommendations SET is_visible = ? WHERE id = ?
	`, visibleInt, recommendationID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// GetAll 获取所有课程推荐（管理员用，包含不可见的）
func GetAll(page, pageSize int, status string, courseName string, teacherName string) ([]model.CourseRecommendation, int64, error) {
	db := database.GetCourseRecDB()
	if db == nil {
		return nil, 0, errors.New("数据库连接失败")
	}

	// 构建查询条件
	var conditions []string
	var args []interface{}

	if status == "pending" {
		conditions = append(conditions, "is_visible = 0")
	} else if status == "approved" {
		conditions = append(conditions, "is_visible = 1")
	}

	if courseName != "" {
		conditions = append(conditions, "course_name LIKE ?")
		args = append(args, "%"+courseName+"%")
	}

	if teacherName != "" {
		conditions = append(conditions, "teacher_name LIKE ?")
		args = append(args, "%"+teacherName+"%")
	}

	// 构建 WHERE 子句
	whereSQL := ""
	if len(conditions) > 0 {
		whereSQL = "WHERE " + joinConditions(conditions, " AND ")
	}

	// 1. 获取总数
	var total int64
	countQuery := "SELECT COUNT(*) FROM course_recommendations " + whereSQL
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 2. 分页查询
	offset := (page - 1) * pageSize
	query := `
		SELECT id, course_name, teacher_name, recommendation_reason, recommender_nickname, recommendation_time, is_visible, campus, recommendation_year
		FROM course_recommendations
		` + whereSQL + `
		ORDER BY recommendation_time DESC
		LIMIT ? OFFSET ?
	`

	// 分页参数追加到 args 末尾
	queryArgs := append(args, pageSize, offset)

	rows, err := db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []model.CourseRecommendation
	for rows.Next() {
		var r model.CourseRecommendation
		if err := rows.Scan(&r.ID, &r.CourseName, &r.TeacherName, &r.RecommendationReason, &r.RecommenderNickname, &r.RecommendationTime, &r.IsVisible, &r.Campus, &r.RecommendationYear); err != nil {
			continue
		}
		list = append(list, r)
	}

	if list == nil {
		return []model.CourseRecommendation{}, total, nil
	}
	return list, total, nil
}

// joinConditions 将条件用指定分隔符连接
func joinConditions(conditions []string, separator string) string {
	result := ""
	for i, c := range conditions {
		if i > 0 {
			result += separator
		}
		result += c
	}
	return result
}

// Delete 删除课程推荐（管理员用）
func Delete(recommendationID int64) error {
	db := database.GetCourseRecDB()
	if db == nil {
		return errors.New("数据库连接失败")
	}

	result, err := db.Exec(`DELETE FROM course_recommendations WHERE id = ?`, recommendationID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

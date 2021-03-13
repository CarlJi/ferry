package service

import (
	"database/sql"
	"ferry/global/orm"
	"ferry/models/process"
	"ferry/pkg/pagination"

	"github.com/gin-gonic/gin"
)

/*
  @Author : lanyulei
*/

type Ranks struct {
	Name  string `json:"name"`
	Total int    `json:"total"`
}

type Statistics struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

func NewStatistics(startTime string, endTime string) *Statistics {
	return &Statistics{
		StartTime: startTime,
		EndTime:   endTime,
	}
}

// 查询范围统计数据
func (s *Statistics) DateRangeStatistics() (statisticsData map[string][]interface{}, err error) {
	var (
		datetime   string
		total      int
		overs      int
		processing int
		sqlValue   string
		rows       *sql.Rows
	)
	sqlValue = `SELECT
		a.click_date,
		ifnull( b.total, 0 ) AS total,
		ifnull( b.overs, 0 ) AS overs,
		ifnull( b.processing, 0 ) AS processing 
	FROM
		(
		SELECT
			curdate() AS click_date UNION ALL
		SELECT
			date_sub( curdate(), INTERVAL 1 DAY ) AS click_date UNION ALL
		SELECT
			date_sub( curdate(), INTERVAL 2 DAY ) AS click_date UNION ALL
		SELECT
			date_sub( curdate(), INTERVAL 3 DAY ) AS click_date UNION ALL
		SELECT
			date_sub( curdate(), INTERVAL 4 DAY ) AS click_date UNION ALL
		SELECT
			date_sub( curdate(), INTERVAL 5 DAY ) AS click_date UNION ALL
		SELECT
			date_sub( curdate(), INTERVAL 6 DAY ) AS click_date 
		) a
		LEFT JOIN (
		SELECT
			a1.datetime AS datetime,
			a1.count AS total,
			b1.count AS overs,
			c.count AS processing
		FROM
			(
			SELECT
				date( create_time ) AS datetime,
				count(*) AS count 
			FROM
				p_work_order_info 
			GROUP BY
			date( create_time )) a1
			LEFT JOIN (
			SELECT
				date( create_time ) AS datetime,
				count(*) AS count 
			FROM
				p_work_order_info 
			WHERE
				is_end = 1 
			GROUP BY
			date( create_time )) b1 ON a1.datetime = b1.datetime
			LEFT JOIN (
			SELECT
				date( create_time ) AS datetime,
				count(*) AS count 
			FROM
				p_work_order_info 
			WHERE
				is_end = 0 
			GROUP BY
			date( create_time )) c ON a1.datetime = c.datetime 
		) b ON a.click_date = b.datetime order by a.click_date;`
	rows, err = orm.Eloquent.Raw(sqlValue).Rows()
	if err != nil {
		return
	}
	defer func() {
		_ = rows.Close()
	}()
	statisticsData = map[string][]interface{}{}
	for rows.Next() {
		err = rows.Scan(&datetime, &total, &overs, &processing)
		if err != nil {
			return
		}
		statisticsData["datetime"] = append(statisticsData["datetime"], datetime[:10])
		statisticsData["total"] = append(statisticsData["total"], total)
		statisticsData["overs"] = append(statisticsData["overs"], overs)
		statisticsData["processing"] = append(statisticsData["processing"], processing)
	}
	return
}

// 查询工单提交排名
func (s *Statistics) SubmitRanking() (submitRankingData map[string][]interface{}, err error) {
	var (
		userId       int
		username     string
		nickname     string
		rankingCount int
		rows         *sql.Rows
	)

	sqlValue := `SELECT
		creator AS user_id,
		sys_user.username AS username,
		sys_user.nick_name,
		COUNT(*) AS rankingCount 
	FROM
		p_work_order_info
		LEFT JOIN sys_user ON sys_user.user_id = p_work_order_info.creator 
	GROUP BY
		p_work_order_info.creator ORDER BY rankingCount limit 6;`

	rows, err = orm.Eloquent.Raw(sqlValue).Rows()
	if err != nil {
		return
	}
	defer func() {
		_ = rows.Close()
	}()
	submitRankingData = map[string][]interface{}{}
	for rows.Next() {
		err = rows.Scan(&userId, &username, &nickname, &rankingCount)
		if err != nil {
			return
		}
		submitRankingData["userId"] = append(submitRankingData["userId"], userId)
		submitRankingData["username"] = append(submitRankingData["username"], username)
		submitRankingData["nickname"] = append(submitRankingData["nickname"], nickname)
		submitRankingData["rankingCount"] = append(submitRankingData["rankingCount"], rankingCount)
	}

	return
}

// 查询工单数量统计
func (s *Statistics) WorkOrderCount(c *gin.Context) (countList map[string]int, err error) {
	var (
		w      *WorkOrder
		result interface{}
	)
	countList = make(map[string]int)
	for _, i := range []int{1, 2, 3, 4} {
		w = NewWorkOrder(i, c)
		if i != 1 {
			result, err = w.PureWorkOrderList()
			if err != nil {
				return
			}
		} else {
			w = NewWorkOrder(i, c)
			result, err = w.WorkOrderList()
			if err != nil {
				return
			}
		}

		if i == 1 {
			countList["upcoming"] = result.(*pagination.Paginator).TotalCount
		} else if i == 2 {
			countList["my_create"] = result.(*pagination.Paginator).TotalCount
		} else if i == 3 {
			countList["related"] = result.(*pagination.Paginator).TotalCount
		} else if i == 4 {
			countList["all"] = result.(*pagination.Paginator).TotalCount
		}
	}

	return
}

// 查询指定范围内的提交工单排名数据
func (s *Statistics) WorkOrderRanks() (ranks []Ranks, err error) {
	ranks = make([]Ranks, 0)

	err = orm.Eloquent.Model(&process.WorkOrderInfo{}).
		Joins("left join p_process_info on p_process_info.id = p_work_order_info.process").
		Select("p_process_info.name as name, count(p_work_order_info.id) as total").
		Where("p_work_order_info.create_time between ? and ?", s.StartTime, s.EndTime).
		Group("p_work_order_info.process").
		Order("total desc").
		Limit(10).
		Scan(&ranks).Error

	return
}

// Database connection

package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/robfig/cron/v3"
	"github.com/xuri/excelize/v2"
)

func main() {
	timer()
	autoCalculateSquad()
	selectIntoExcel()
}

func timer() {
	msc, _ := time.LoadLocation("Europe/Moscow")
	c := cron.New(cron.WithLocation(msc))

	c.AddFunc("@every 1m", func() {
		deleteFromParentsTimer()
	})

	c.Start()

	for {
		time.Sleep(time.Second * 1)
	}
}

func deleteFromParentsTimer() {
	connStr := "user=postgres password=6858 dbname=test_1 sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	type dateReg struct {
		id        int
		dateR     time.Time
		statusPay bool
	}

	rows, err := db.Query("select id, date_reg, status_pay from parents")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	dateRegs := []dateReg{}

	for rows.Next() {
		d := dateReg{}
		err := rows.Scan(&d.id, &d.dateR, &d.statusPay)
		if err != nil {
			log.Fatal(err)
		}
		dateRegs = append(dateRegs, d)
	}

	for _, d := range dateRegs {
		if !d.statusPay {
			differDate := time.Since(d.dateR)
			fmt.Println(differDate)

			if differDate.Hours() > 720 {
				result, err := db.Exec("delete from parents where id = $1", d.id)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println(result.RowsAffected())
			} else {
				fmt.Println("payment deadline has not expired")
			}
		}
	}
}

func autoCalculateSquad() {
	connStr := "user=postgres password=6858 dbname=test_1 sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	type childCount struct {
		count int
	}

	row := db.QueryRow("select * from child_count")
	childC := childCount{}
	err = row.Scan(&childC.count)
	if err != nil {
		panic(err)
	}

	totalPlace := childC.count
	squadPlace := 25
	i := 1

	if totalPlace%squadPlace != 0 {
		for ; i <= totalPlace/squadPlace; i++ {
			_, err := db.Exec("insert into groups (group_year, group_number, count) values (0, $1, $2)", i, squadPlace)
			if err != nil {
				panic(err)
			}
		}

		_, err := db.Exec("insert into groups (group_year, group_number, count) values (0, $1, $2)", i, totalPlace%squadPlace)
		if err != nil {
			panic(err)
		}
		i = 0
	} else {
		for ; i <= totalPlace/squadPlace; i++ {
			_, err := db.Exec("insert into groups (group_year, group_number, count) values (0, $1, $2)", i, squadPlace)
			if err != nil {
				panic(err)
			}
		}
		i = 0
	}
}

func selectIntoExcel() {
	connStr := "user=postgres password=6858 dbname=test_1 sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	type data struct {
		surnameChildren string
		nameChildren    string
		seriesPass      string
		numberPass      string
		surnameParent   string
		nameParent      string
		phoneParent     string
		statusPay       bool
		groupNumber     int
	}

	rows, err := db.Query("select children.surname, children.name, passport.series, passport.number, parents.surname, parents.name, parents.phone, parents.status_pay, groups.group_number from children  join passport on passport.id = children.id join ticket on ticket.id = children.id join parents on parents.id = children.id_parent join groups on groups.id = children.id_group")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	moreData := []data{}

	for rows.Next() {
		d := data{}
		err := rows.Scan(&d.surnameChildren, &d.nameChildren, &d.seriesPass, &d.numberPass, &d.surnameParent, &d.nameParent, &d.phoneParent, &d.statusPay, &d.groupNumber)
		if err != nil {
			log.Fatal(err)
		}
		moreData = append(moreData, d)
	}

	f := excelize.NewFile()

	index := f.NewSheet("Sheet1")

	f.SetCellValue("Sheet1", "A1", "Фамилия ребенка")
	f.SetCellValue("Sheet1", "B1", "Имя ребенка")
	f.SetCellValue("Sheet1", "C1", "Серия паспорта ребенка")
	f.SetCellValue("Sheet1", "D1", "Номер паспорта ребенка")
	f.SetCellValue("Sheet1", "E1", "Фамиляи представителя")
	f.SetCellValue("Sheet1", "F1", "Имя представителя")
	f.SetCellValue("Sheet1", "G1", "Телефон представилетеля")
	f.SetCellValue("Sheet1", "H1", "Статус оплаты")
	f.SetCellValue("Sheet1", "I1", "Отряд ребенка")

	i := 2

	for _, d := range moreData {
		f.SetCellValue("Sheet1", fmt.Sprintf("A%d", i), d.surnameChildren)
		f.SetCellValue("Sheet1", fmt.Sprintf("B%d", i), d.nameChildren)
		f.SetCellValue("Sheet1", fmt.Sprintf("C%d", i), d.seriesPass)
		f.SetCellValue("Sheet1", fmt.Sprintf("D%d", i), d.numberPass)
		f.SetCellValue("Sheet1", fmt.Sprintf("E%d", i), d.surnameParent)
		f.SetCellValue("Sheet1", fmt.Sprintf("F%d", i), d.nameParent)
		f.SetCellValue("Sheet1", fmt.Sprintf("G%d", i), d.phoneParent)
		f.SetCellValue("Sheet1", fmt.Sprintf("H%d", i), d.statusPay)
		f.SetCellValue("Sheet1", fmt.Sprintf("I%d", i), d.groupNumber)
		i++
	}

	i = 2

	f.SetActiveSheet(index)

	error := f.SaveAs("C:/Users/Carlo/Desktop/test.xlsx")
	if error != nil {
		log.Fatal(err)
	}
}

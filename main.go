package main

import (
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type iblock struct {
	Id        int    `json:"id" gorm:"primaryKey"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Parent_id int    `json:"parent_id"`
	Left      int    `json:"left"`
	Right     int    `json:"right"`
	Depth     int    `json:"depth"`
}

func (iblock) TableName() string {
	return "iblocks"
}

type iblock_elements struct {
	Id                int                 `json:"id" gorm:"primaryKey"`
	Slug              string              `json:"slug"`
	Name              string              `json:"name"`
	Iblock_prop_value []iblock_prop_value `gorm:"foreignkey:Iblock_elementsID"`
}

func (iblock_elements) TableName() string {
	return "iblock_elements"
}

type iblock_property struct {
	Id                int                 `json:"id" gorm:"primaryKey"`
	Is_number         int                 `json:"is_number"`
	Is_multy          int                 `json:"is_multy"`
	Name              string              `json:"name"`
	IblockId          int                 `gorm:"column:iblock_id;"`
	Iblock_prop_value []iblock_prop_value `gorm:"foreignkey:Iblock_propertyID"`
}

func (iblock_property) TableName() string {
	return "iblock_properties"
}

type iblock_prop_value struct {
	Id                int    `gorm:"primaryKey"`
	Value             string `json:"value"`
	Slug              string `json:"slug"`
	Value_number      int    `json:"value_number"`
	Iblock_propertyID int    `gorm:"column:prop_id;"`
	Iblock_elementsID int    `gorm:"column:el_id;"`
}

func (iblock_prop_value) TableName() string {
	return "iblock_prop_values"
}

// service layer
type _db struct{}

func (_db _db) init() *gorm.DB {
	dsn := "root:root@tcp(127.0.0.1:3306)/iblockcms?charset=utf8mb4&parseTime=True&loc=Local"
	db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	return db
}

type catalog_node struct {
	Childrens []*catalog_node `json:"childrens"`
	Value     iblock          `json:"item"`
}

type createTreeRes struct {
	catalog catalog_node
	ids     []int
}

func createTree(id int) createTreeRes {
	_db := _db{}
	db := _db.init()
	var ids []int
	var deep func(node *catalog_node)
	deep = func(node *catalog_node) {
		ids = append(ids, node.Value.Id)
		var _iblock []iblock
		db.Where("parent_id = ?", node.Value.Id).Find(&_iblock)
		for _, item := range _iblock {
			c := catalog_node{Value: item, Childrens: nil}
			node.Childrens = append(node.Childrens, &c)
			deep(&c)
		}

	}
	var _iblock []iblock
	db.Where("id = ?", id).Find(&_iblock)
	c := catalog_node{Childrens: nil, Value: _iblock[0]}
	ids = append(ids, c.Value.Id)
	deep(&c)
	return createTreeRes{
		catalog: c,
		ids:     ids,
	}
}
func getElements(ids []int, page int) []iblock_elements {
	_db := _db{}
	db := _db.init()
	var elements []iblock_elements
	db.Offset(page*5).Limit(5).Preload("Iblock_prop_value").Where("iblock_id", ids).Find(&elements)
	return elements
}

//

func main() {
	app := fiber.New()
	app.Get("/catalog/:page?", func(c *fiber.Ctx) error {
		page := c.Params("page")
		q := createTree(1)
		catalog := q.catalog
		i, _ := strconv.Atoi(page)
		els := getElements(q.ids, i)

		return c.JSON(&fiber.Map{
			"success": true,
			"catalog": catalog,
			"els":     els,
		})
	})

	log.Fatal(app.Listen(":3000"))

}

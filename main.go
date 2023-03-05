package main

import (
	"log"
	"strconv"
	"sync"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type iblock struct {
	Id              int               `json:"id" gorm:"primaryKey"`
	Name            string            `json:"name"`
	Slug            string            `json:"slug"`
	Parent_id       int               `json:"parent_id"`
	Left            int               `json:"left"`
	Right           int               `json:"right"`
	Depth           int               `json:"depth"`
	Iblock_property []iblock_property `json:"properties" gorm:"foreignkey:IblockId"`
}

func (iblock) TableName() string {
	return "iblocks"
}

type iblock_elements struct {
	Id                int                 `json:"id" gorm:"primaryKey"`
	Slug              string              `json:"slug"`
	Name              string              `json:"name"`
	Iblock_id         int                 `json:"iblock_id" gorm:"column:iblock_id;"`
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
	Id                int             `gorm:"primaryKey"`
	Value             string          `json:"value"`
	Slug              string          `json:"slug"`
	Value_number      int             `json:"value_number"`
	Iblock_propertyID int             `gorm:"column:prop_id;"`
	Iblock_elementsID int             `gorm:"column:el_id;"`
	Iblock_property   iblock_property `json:"prop"`
}

func (iblock_prop_value) TableName() string {
	return "iblock_prop_values"
}

// service layer
type _db struct {
	_instance *gorm.DB
}

var __db = _db{}

func (_db *_db) init() *gorm.DB {
	if _db._instance != nil {
		return _db._instance
	}
	dsn := "root:root@tcp(127.0.0.1:3306)/iblockcms?charset=utf8mb4&parseTime=True&loc=Local"
	_db._instance, _ = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	return _db._instance
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
	db := __db.init()
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

type or_arr struct {
	value   []string
	prop_id int
}
type range_arr struct {
	value_to   int
	value_from int
	prop_id    int
}

func getElements(range_arr []range_arr, or []or_arr, ids []int, page int) []iblock_elements {
	db := __db.init()

	//gorm not support WhereHas(laravel) https://github.com/go-gorm/gorm/issues/3871
	c_name := 0
	params := make(map[string]interface{})
	params["name"+strconv.Itoa(c_name)] = ids
	filterStr := "select * from `iblock_elements` where `iblock_id` in @name" + strconv.Itoa(c_name) + " and `name` != 'op'"
	c_name++
	c1 := 0
	c2 := 0
	if len(or) > 0 {
		for _, item := range or {

			filterStr += " and exists (select * from `iblock_prop_values` where `iblock_elements`.`id` = `iblock_prop_values`.`el_id` and `prop_id` = @name" + strconv.Itoa(c_name) + " and ("
			params["name"+strconv.Itoa(c_name)] = item.prop_id
			c_name++
			c2 = 0
			for _, val := range item.value {
				if c2 != 0 {
					filterStr += " or "
				}
				filterStr += "`value` = " + "@name" + strconv.Itoa(c_name)
				params["name"+strconv.Itoa(c_name)] = val
				c_name++
				c2++
			}
			filterStr += "))"
			c1++
		}
	}

	if len(range_arr) > 0 {
		for _, item := range range_arr {

			filterStr += " and exists (select * from `iblock_prop_values` where `iblock_elements`.`id` = `iblock_prop_values`.`el_id` and `prop_id` = @name" + strconv.Itoa(c_name) + " and ("
			params["name"+strconv.Itoa(c_name)] = item.prop_id
			c_name++
			c2 = 0

			filterStr += "`value_number` >= @name" + strconv.Itoa(c_name) + " and"
			params["name"+strconv.Itoa(c_name)] = item.value_from

			c_name++
			filterStr += " `value_number` <= @name" + strconv.Itoa(c_name) + " "
			params["name"+strconv.Itoa(c_name)] = item.value_to
			c_name++

			filterStr += "))"
			c1++
		}
	}

	filterStr += " LIMIT 5 OFFSET " + "@name" + strconv.Itoa(c_name)
	params["name"+strconv.Itoa(c_name)] = strconv.Itoa((page - 1) * 5)

	var elements []iblock_elements
	var res []iblock_elements
	var hack []int
	db.Raw(filterStr, params).Scan(&elements)
	for _, item := range elements {
		hack = append(hack, item.Id)
	}
	db.Preload("Iblock_prop_value.Iblock_property").Where("id", hack).Find(&res)
	return res
}

var properties_cache = make(map[int]map[int][]iblock_prop_value)

func getProperties(id int) map[int][]iblock_prop_value {
	db := __db.init()
	if properties_cache[id] != nil {
		return properties_cache[id]
	}
	res := make(map[int][]iblock_prop_value)
	var props []iblock_property
	var deep func(id int)
	deep = func(id int) {
		var _iblock []iblock
		db.Preload("Iblock_property").Where("id = ?", id).Find(&_iblock)
		for _, item := range _iblock {
			for _, p := range item.Iblock_property {
				props = append(props, p)
			}
		}
		for _, item := range _iblock {
			deep(item.Parent_id)
		}
	}
	deep(id)

	var wg sync.WaitGroup
	wg.Add(len(props))

	var thread func(item iblock_property)
	thread = func(item iblock_property) {
		var _props []iblock_prop_value
		db.Where("prop_id = ?", item.Id).Group("value").Find(&_props)
		for _, p := range _props {
			res[item.Id] = append(res[item.Id], p)
		}
		defer wg.Done()
	}

	for _, item := range props {
		go thread(item)
	}
	wg.Wait()
	properties_cache[id] = res
	return res
}

//

func main() {
	app := fiber.New()
	app.Get("/catalog/:id/:page/", func(c *fiber.Ctx) error {
		id, _ := strconv.Atoi(c.Params("id"))
		page, _ := strconv.Atoi(c.Params("page"))
		q := createTree(id)
		catalog := q.catalog
		//var or = []or_arr{{value: []string{"геймерская", "обычная"}, prop_id: 2}, {value: []string{"внутренняя"}, prop_id: 3}}
		//var range_arr = []range_arr{{value_to: 900, value_from: 0, prop_id: 15}}
		var or = []or_arr{}
		var range_arr = []range_arr{}
		els := getElements(range_arr, or, q.ids, page)
		props := getProperties(id)
		return c.JSON(&fiber.Map{
			"catalog": catalog,
			"els":     els,
			"props":   props,
		})
	})

	log.Fatal(app.Listen(":3000"))

}

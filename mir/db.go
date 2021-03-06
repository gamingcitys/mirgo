package mir

import (
	"github.com/jinzhu/gorm"
	"github.com/yenkeia/mirgo/common"
)

type DB struct {
	db *gorm.DB
}

// adb 保存游戏运行时生成的数据
var adb *DB

func NewDB(db *gorm.DB) *DB {

	r := &DB{db: db}

	db.SingularTable(true)
	db.AutoMigrate(
		&common.Account{},
		&common.AccountCharacter{},
		&common.Character{},
		&common.CharacterUserItem{},
		&common.UserItem{},
		&common.UserMagic{},
	)

	return r
}

func (d *DB) Table(name string) *gorm.DB {
	return d.db.Table(name)
}

func (d *DB) setCharacterAttr(p *Player, attr string, value interface{}) {
	d.db.Table("character").Where("id = ?", p.ID).Update(attr, value)
}

func (d *DB) SyncLevel(p *Player) {
	d.setCharacterAttr(p, "level", p.Level)
}

func (d *DB) SyncGold(p *Player) {
	d.setCharacterAttr(p, "gold", p.Gold)
}

func (d *DB) SyncPosition(p *Player) {
	d.setCharacterAttr(p, "current_map_id", p.Map.Info.ID)
	d.setCharacterAttr(p, "direction", p.GetDirection())
	d.setCharacterAttr(p, "current_location_x", p.GetPoint().X)
	d.setCharacterAttr(p, "current_location_y", p.GetPoint().Y)
}

package mir

import (
	"fmt"
	"time"

	"github.com/yenkeia/mirgo/common"
	"github.com/yenkeia/mirgo/ut"
)

// Map ...
type Map struct {
	Width  int
	Height int
	Info   *common.MapInfo
	cells  []*Cell

	players        map[uint32]*Player
	monsters       map[uint32]*Monster
	npcs           map[uint32]*NPC
	activedObjects map[uint32]IProcessObject

	ActionList *ActionList
}

func NewMap(w, h int) *Map {
	m := &Map{
		Width:          w,
		Height:         h,
		cells:          make([]*Cell, w*h),
		players:        map[uint32]*Player{},
		monsters:       map[uint32]*Monster{},
		npcs:           map[uint32]*NPC{},
		activedObjects: map[uint32]IProcessObject{},
		ActionList:     NewActionList(),
	}
	return m
}

func (m *Map) AddActiveObj(o interface{}) {
	v := o.(IProcessObject)
	m.activedObjects[v.GetID()] = v
}

func (m *Map) DelActiveObj(o interface{}) {
	delete(m.activedObjects, o.(IProcessObject).GetID())
}

func (m *Map) Frame(dt time.Duration) {

	m.ActionList.Execute()

	for _, p := range m.players {
		p.Process(dt)
	}

	for _, o := range m.activedObjects {
		o.Process(dt)
	}

	// for _, monster := range m.monsters {
	// 	if monster.GetPlayerCount() > 0 {
	// 		monster.Process(dt)
	// 	}
	// }
	// for _, npc := range m.npcs {
	// 	if npc.GetPlayerCount() > 0 {
	// 		npc.Process(dt)
	// 	}
	// }
}

func (m *Map) GetCell(p common.Point) *Cell {
	return m.GetCellXY(int(p.X), int(p.Y))
}
func (m *Map) GetCellXY(x, y int) *Cell {
	if m.InMap(x, y) {
		return m.cells[x+y*m.Width]
	} else {
		return nil
	}
}

func (m *Map) InMap(x, y int) bool {
	return x >= 0 && x < m.Width && y >= 0 && y < m.Height
}

func (m *Map) ValidPoint(p common.Point) bool {
	c := m.GetCell(p)
	return c != nil && c.IsValid()
}

func (m *Map) SetCell(p common.Point, c *Cell) {
	m.SetCellXY(int(p.X), int(p.Y), c)
}
func (m *Map) SetCellXY(x, y int, c *Cell) {
	m.cells[x+y*m.Width] = c
}

func (m *Map) String() string {
	return fmt.Sprintf("Map(%d, Filename: %s, Title: %s, Width: %d, Height: %d)", m.Info.ID, m.Info.Filename, m.Info.Title, m.Width, m.Height)
}

func (m *Map) GetAllPlayers() map[uint32]*Player {
	return m.players
}
func (m *Map) GetNPC(id uint32) *NPC {
	return m.npcs[id]
}

// Broadcast send message to all players in this map
func (m *Map) Broadcast(msg interface{}) {
	for _, p := range m.players {
		p.Enqueue(msg)
	}
}

// 位置，消息，跳过玩家
func (m *Map) BroadcastP(pos common.Point, msg interface{}, me *Player) {
	// m.Submit(NewTask(func(args ...interface{}) {
	for _, plr := range m.players {
		if InRange(pos, plr.CurrentLocation, DataRange) {
			if plr != me {
				plr.Enqueue(msg)
			}
		}
	}
	// }))
}

func (m *Map) AddObject(obj IMapObject) (string, bool) {
	if obj == nil || obj.GetID() == 0 {
		return "", false
	}
	c := m.GetCell(obj.GetPoint())
	if c == nil {
		// FIXME
		return fmt.Sprintf("pos: %s is not walkable\n", obj.GetPoint()), false
	}
	c.AddObject(obj)

	switch obj.(type) {
	case *Player:
		m.players[obj.GetID()] = obj.(*Player)
	case *NPC:
		m.npcs[obj.GetID()] = obj.(*NPC)
	case *Monster:
		m.monsters[obj.GetID()] = obj.(*Monster)
	}

	return "", true
}

func (m *Map) DeleteObject(obj IMapObject) {
	if obj == nil || obj.GetID() == 0 {
		return
	}
	c := m.GetCell(obj.GetPoint())
	if c == nil {
		return
	}
	c.DeleteObject(obj)

	delete(m.activedObjects, obj.GetID())

	switch obj.(type) {
	case *Player:
		delete(m.players, obj.GetID())
	case *Monster:
		delete(m.monsters, obj.GetID())
	case *NPC:
		delete(m.npcs, obj.GetID())
	}
}

// UpdateObject 更新对象在 Cells, AOI 中的数据, 如果更新成功返回 true
func (m *Map) UpdateObject(obj IMapObject, point common.Point) bool {
	destcell := m.GetCell(point)
	if destcell == nil || !destcell.CanWalk() {
		return false
	}

	for _, o := range destcell.objects {
		if o.IsBlocking() {
			return false
		}
	}

	sourcecell := obj.GetCell()
	sourcecell.DeleteObject(obj)
	destcell.AddObject(obj)

	switch obj.GetRace() {
	case common.ObjectTypePlayer:
		p := obj.(*Player)
		p.Broadcast(ServerMessage{}.ObjectPlayer(p))
		p.EnqueueAreaObjects(sourcecell, destcell)
	case common.ObjectTypeMonster:
		m := obj.(*Monster)
		m.Broadcast(ServerMessage{}.ObjectMonster(m))
	}

	return true
}

// InitNPCs 初始化地图上的 NPC
func (m *Map) InitNPCs() error {
	for _, ni := range data.NpcInfos {
		ni := ni
		if ni.MapID == m.Info.ID {
			n := NewNPC(m, env.NewObjectID(), ni)
			m.AddObject(n)
		}
	}
	return nil
}

// InitMonsters 初始化地图上的怪物
func (m *Map) InitMonsters() error {
	for _, ri := range data.RespawnInfos {
		ri := ri
		if ri.MapID == m.Info.ID {
			cnt := ri.Count
			for i := 0; i < cnt; i++ {
				p, err := m.GetValidPoint(ri.LocationX, ri.LocationY, ri.Spread)
				if err != nil {
					continue
				}
				m.AddObject(NewMonster(m, p, data.GetMonsterInfoByID(ri.MonsterID)))
			}
		}
	}
	return nil
}

func (m *Map) OpenDoor(doorindex byte) bool {
	// TODO
	return true
}

func (m *Map) CheckDoorOpen(loc common.Point) bool {
	// TODO
	return true
}

// GetValidPoint ...
func (m *Map) GetValidPoint(x int, y int, spread int) (common.Point, error) {
	if spread == 0 {
		c := m.GetCellXY(x, y)
		if c != nil && c.CanWalk() && !c.HasObject() {
			return c.Point, nil
		}
		return common.Point{}, fmt.Errorf("GetValidPoint: (x: %d, y: %d), spread: %d\n", x, y, spread)
	}

	for i := 0; i < 500; i++ {
		p := common.Point{
			X: uint32(ut.AbsInt(x + ut.RandomInt(-spread, spread))),
			Y: uint32(ut.AbsInt(y + ut.RandomInt(-spread, spread))),
		}
		c := m.GetCell(p)
		if c == nil || !c.CanWalk() {
			continue
		}
		return p, nil
	}
	return common.Point{}, fmt.Errorf("map(%v) no valid point in (%d,%d) spread: %d", m, x, y, spread)
}

func (m *Map) GetNextCell(cell *Cell, direction common.MirDirection, step uint32) *Cell {
	p := cell.Point.NextPoint(direction, step)
	return m.GetCell(p)
}

func (m *Map) GetObjectInAreaByID(id uint32, p common.Point) IMapObject {
	var ret IMapObject
	m.RangeObject(p, 1, func(o IMapObject) bool {
		if o.GetID() == id {
			ret = o
			return false
		}
		return true
	})

	return ret
}

// 从p点开始（包含P），由内至外向周围遍历cell。回调函数返回false，停止遍历
func (m *Map) RangeCell(p common.Point, depth int, fun func(c *Cell, x, y int) bool) {

	px, py := int(p.X), int(p.Y)

	for d := 0; d <= depth; d++ {
		for y := py - d; y <= py+d; y++ {
			if y < 0 {
				continue
			}
			if y >= m.Height {
				break
			}

			for x := px - d; x <= px+d; {

				if x >= m.Width {
					break
				}

				if x >= 0 {
					if !fun(m.GetCellXY(x, y), x, y) {
						return
					}
				}

				if y-py == d || y-py == -d {
					x++ // x += 1
				} else {
					x += d * 2
				}
			}
		}
	}
}

func (m *Map) RangeObject(p common.Point, depth int, fun func(IMapObject) bool) {
	var ret = true
	m.RangeCell(p, depth, func(c *Cell, _, _ int) bool {
		if c != nil && c.objects != nil {
			for _, o := range c.objects {
				ret = fun(o)
				if ret == false {
					return false
				}
			}
		}

		return ret
	})
}

// Cell集合
type CellSet struct {
	M map[*Cell]bool
}

func NewCellSet() *CellSet {
	return &CellSet{M: map[*Cell]bool{}}
}

func (c *CellSet) Add(m *Map, x, y int, b bool) {
	cell := m.GetCellXY(x, y)
	if cell != nil {
		c.M[cell] = b
	}
}

// 根据两个点，求出 远离datarange内的cell，和新加的cell
func (m *Map) CalcDiff(from, to common.Point, datarange int) *CellSet {
	fx, fy, tx, ty := int(from.X), int(from.Y), int(to.X), int(to.Y)

	xChange, yChange := tx-fx, ty-fy
	set := NewCellSet()

	if xChange > 0 { // 右移
		for x := 0; x < xChange; x++ {
			for y := fy - datarange; y <= fy+datarange; y++ {
				set.Add(m, fx-datarange+x, y, false) // 左
			}
			for y := ty - datarange; y <= ty+datarange; y++ {
				set.Add(m, tx+datarange-x, y, true) // 右
			}
		}
	} else { // 左移
		for x := 0; x > xChange; x-- {
			for y := ty - datarange; y <= ty+datarange; y++ {
				set.Add(m, tx-datarange-x, y, true) // 左
			}
			for y := fy - datarange; y <= fy+datarange; y++ {
				set.Add(m, fx+datarange+x, y, false) // 右
			}
		}
	}
	if yChange < 0 { // 上移
		for y := 0; y > yChange; y-- {
			for x := tx - datarange; x <= tx+datarange; x++ {
				set.Add(m, x, ty-datarange-y, true) // 上
			}
			for x := fx - datarange; x <= fx+datarange; x++ {
				set.Add(m, x, fy+datarange+y, false) // 下
			}
		}
	} else { // 下移
		for y := 0; y < yChange; y++ {
			for x := fx - datarange; x <= fx+datarange; x++ {
				set.Add(m, x, fy-datarange+y, false) // 上
			}
			for x := tx - datarange; x <= tx+datarange; x++ {
				set.Add(m, x, ty+datarange-y, true) // 下
			}
		}
	}

	return set
}

// CompleteMagic ...
func (m *Map) CompleteMagic(args ...interface{}) {
	magic := args[0].(*common.UserMagic)
	switch magic.Spell {
	case common.SpellSummonSkeleton, common.SpellSummonShinsu, common.SpellSummonHolyDeva, common.SpellSummonVampire, common.SpellSummonToad, common.SpellSummonSnakes:
		player := args[1].(*Player)
		monster := args[2].(*Monster)
		front := args[3].(common.Point)
		if monster.Master.IsDead() {
			return
		}
		cell := m.GetCell(front)
		if cell.IsValid() {
			monster.Master = player
			player.Pets = append(player.Pets, monster)
			monster.Spawn()
		}
	case common.SpellMassHealing:
		value := args[1].(int)
		location := args[2].(common.Point)
		player := args[3].(*Player)
		m.RangeObject(location, 1, func(o IMapObject) bool {
			if o.GetRace() == common.ObjectTypePlayer && o.IsFriendlyTarget(player) {
				target := o.(*Player)
				for i := range target.Buffs {
					if target.Buffs[i].BuffType == common.BuffTypeHiding {
						return true
					}
				}
				target.AddBuff(NewBuff(player.NewObjectID(), common.BuffTypeHiding, 0, time.Now().Add(time.Duration(value*1000)*time.Millisecond)))
			}
			return true
		})
	case common.SpellSoulShield, common.SpellBlessedArmour:
		value := args[1].(int)
		location := args[2].(common.Point)
		player := args[3].(*Player)
		buffType := common.BuffTypeSoulShield
		if magic.Spell == common.SpellBlessedArmour {
			buffType = common.BuffTypeBlessedArmour
		}
		m.RangeObject(location, 1, func(o IMapObject) bool {
			if o.GetRace() == common.ObjectTypePlayer {
				target := o.(*Player)
				target.AddBuff(NewBuff(player.NewObjectID(), buffType, int(target.Level)/7+4, time.Now().Add(time.Duration(value*1000)*time.Millisecond)))
			}
			return true
		})
	case common.SpellFireWall:
		// player := args[1].(*Player)
		// value := args[2].(int)
		// location := args[3].(common.Point)
		// player.LevelMagic(magic)
		// TODO SpellObject
	case common.SpellLightning:
		// player := args[1].(*Player)
		// value := args[2].(int)
		// location := args[3].(common.Point)
		// direction := args[4].(common.MirDirection)
	case common.SpellThunderStorm, common.SpellFlameField, common.SpellNapalmShot, common.SpellStormEscape:
		// player := args[1].(*Player)
		// value := args[2].(int)
		// location := args[3].(common.Point)
	}
}

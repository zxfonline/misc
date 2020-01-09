package match

import (
	//	"log"
	"sort"
	"time"
)

/**
构建线性自扩散的区间节点
point              int   //中心点
baseRadius             int   //初始半径
incrPerSecond      int   //每秒扩散的半径
maxWaitTime      int   //最大扩散时间 秒
insertTime           int   //插入时间戳
*/
func NewLineNode(point, baseRadius, incrPerSecond, maxWaitTime, insertTime int) *LineNode {
	if insertTime == 0 {
		insertTime = time.Now().Second()
	}
	return &LineNode{
		point:         point,
		baseRadius:    baseRadius,
		incrPerSecond: incrPerSecond,
		maxWaitTime:   maxWaitTime,
		insertTime:    insertTime,
		updaterIdx:    -1,
	}
}

type LineNode struct {
	point         int //中心点
	baseRadius    int //初始半径
	incrPerSecond int //每秒扩散的半径
	maxWaitTime   int //最大扩散时间 秒
	insertTime    int //加入时间 秒
	updaterIdx    int //动态下标(仅用于内部迭代查找范围快速定位)
	ExpireTime    int //过期截止时间 秒 *(!=0则会判断过期)*
	//---------
	Id          int //外部控制的唯一id，主键
	NeighborMax int // PLAYER=需要的最多邻居
	NeighborMin int // PLAYER=需要的最小邻居
	//多种类型|操作数值
	SubType int
}

//根据传入的时间戳(秒),判断是否过期(主要用于房间锁定后或者人数满了)
func (nd *LineNode) ExpireTimed(secondOfNow int) bool {
	return nd.ExpireTime != 0 && secondOfNow >= nd.ExpireTime
}

//根据传入的时间戳(秒),获取当前节点的半径
func (nd *LineNode) CurrtRadius(secondOfNow int) int {
	if nd.incrPerSecond == 0 {
		return nd.baseRadius
	} else {
		second := secondOfNow - nd.insertTime
		if second >= nd.maxWaitTime {
			second = nd.maxWaitTime
		} else if second < 0 {
			second = 0
		}
		return nd.baseRadius + nd.incrPerSecond*second
	}
}

//节点是否等待
func (nd *LineNode) WaitFull(secondOfNow int) bool {
	return secondOfNow-nd.insertTime >= nd.maxWaitTime
}

//节点最大扩散半径
func (nd *LineNode) MaxRadius() int {
	if nd.incrPerSecond == 0 {
		return nd.baseRadius
	} else {
		return nd.baseRadius + nd.incrPerSecond*nd.maxWaitTime
	}
}

//节点中心点
func (nd *LineNode) Point() int {
	return nd.point
}

//节点初始半径
func (nd *LineNode) BaseRadius() int {
	return nd.baseRadius
}

//节点每秒扩散的半径
func (nd *LineNode) IncrPerSecond() int {
	return nd.incrPerSecond
}

//节点初始时间
func (nd *LineNode) InsertTime() int {
	return nd.insertTime
}

//节点最大扩散时间 秒
func (nd *LineNode) MaxWaitTime() int {
	return nd.maxWaitTime
}

//构建线性自扩散的区间节点列表(*非线程安全*,建议单线程使用)
func NewLineNodeList() *LineNodeList {
	return &LineNodeList{}
}

type LineNodeList struct {
	entries   []*LineNode
	maxRadius int //列表中节点的最大半径
}

func (nd *LineNodeList) Len() int {
	return len(nd.entries)
}

func (nd *LineNodeList) AddNode(node *LineNode) {
	node.updaterIdx = -1
	if tRadius := node.MaxRadius(); tRadius > nd.maxRadius {
		nd.maxRadius = tRadius
	}
	if len(nd.entries) == 0 {
		nd.entries = append(nd.entries, node)
	} else {
		size := nd.Len()
		index := sort.Search(size, func(i int) bool { return nd.entries[i].point > node.point })

		if index < size { //插入
			tt := make([]*LineNode, size+1)
			copy(tt, nd.entries[:index])
			tt[index] = node
			copy(tt[index+1:], nd.entries[index:])
			nd.entries = tt
		} else { //新增
			nd.entries = append(nd.entries, node)
		}
	}
}

/**
移除队列中的子节点,返回移除的节点的下标 -1:没找到,[0,len)
*/
func (nd *LineNodeList) RemoveNode(node *LineNode) int {
	search := true
	if node.updaterIdx != -1 { //外部定位过坐标，做一次检查，没有则进行搜索
		if node.updaterIdx < nd.Len() && nd.entries[node.updaterIdx] == node {
			search = false
		}
	}
	var found bool
	var index int
	if search {
		found, index = nd.FindNode(node)
		if found {
			node.updaterIdx = -1
		}
	} else {
		found = true
		index = node.updaterIdx
		node.updaterIdx = -1
	}
	if found {
		if index == 0 {
			if len(nd.entries) == 1 {
				nd.entries = nd.entries[:0]
			} else {
				nd.entries = nd.entries[1:]
			}
		} else if index == len(nd.entries)-1 {
			nd.entries = nd.entries[:index]
		} else {
			nd.entries = append(nd.entries[:index], nd.entries[index+1:]...)
		}
		return index
	} else {
		return -1
	}
}

/**
迭代列表中所有节点
startIndex:迭代开始下标
cursor:迭代回调函数
   node :元素节点
   crtIndex:node所在列表下标
   continuz:是否继续迭代
*/
func (nd *LineNodeList) Iterator(startIndex int, cursor func(*LineNode, int) bool) {
	for index, n := startIndex, nd.Len(); index < n; index++ {
		node := nd.entries[index]
		node.updaterIdx = index //方便 SearchRange 快速定位
		if continuz := cursor(node, index); !continuz {
			return
		}
	}
}

/**
迭代列表中所有节点
   secondOfNow:当前时间 秒
   cpoint:基础元素节点
   needNum:最多需要的邻居数(>0表示找出指定所有,<=0有多少找多少)
   return:
      neighbor:有交集的邻居节点
*/
func (nd *LineNodeList) SearchRange(secondOfNow int, cpoint *LineNode, needNum int) (neighbor []*LineNode) {
	if needNum <= 0 {
		return
	}
	// 顺序查找交集节点
	nowRadius := cpoint.CurrtRadius(secondOfNow) //基础节点当前半径
	maxRadius := nowRadius + nd.maxRadius        //总共查找半径
	cindex := cpoint.updaterIdx                  //基础元素所在列表中下标
	if nd.maxRadius != 0 && cpoint.MaxRadius() > nd.maxRadius {
		maxRadius = nowRadius + cpoint.MaxRadius()
	}
	if cindex != -1 { //外部定位过坐标，做一次检查，没有则进行搜索
		if !(cindex < nd.Len() && nd.entries[cindex] == cpoint) {
			cindex = -1
		}
	}
	if cindex < 0 {
		cindex = nd.SearchNeighbor(cpoint)
	}
	if cindex > nd.Len()-1 {
		cindex = nd.Len() - 1
	}
	//	log.Println("cindex=", cindex, cpoint.Id)
	//先左
	for i := cindex; i >= 0; i-- {
		//		log.Println("index=", i)
		check := nd.entries[i]
		//		log.Println("matching room", check.Id)
		if check == cpoint || check.ExpireTimed(secondOfNow) {
			//			log.Println("skip match room", check.Id)
			continue
		}
		if check.SubType&cpoint.SubType == 0 {
			//			log.Println("skip match room", check.Id)
			continue
		}
		distance := cpoint.point - check.point //两点之间的距离
		if distance < 0 {
			distance = -distance
		}
		matchRadius := nowRadius + check.CurrtRadius(secondOfNow) //邻居间产生交集最大距离
		//		log.Printf("l 玩家半径:%v,房间半径：%v,最大半径:%v,玩家中心点：%v,房间中心点：%v,两点直线距离：%v\n", nowRadius, check.CurrtRadius(secondOfNow), matchRadius, cpoint.point, check.point, distance)
		if distance <= matchRadius { //有交集#
			//			log.Printf("l 房间信息：%+v，玩家信息：%+v\n", check, cpoint)
			neighbor = append(neighbor, check)
			check.updaterIdx = i
			if needNum > 0 && len(neighbor) >= needNum {
				//				log.Println("xx", check.Id)
				return
			}
			//			log.Println("ok", check.Id)
		} else if matchRadius >= maxRadius { //超出了最大查找范围无需再向边缘查找
			//			log.Println("not match room", check.Id)
			break
		}
	}
	//后右
	for i, n := cindex+1, nd.Len(); i < n; i++ {
		check := nd.entries[i]
		if check.ExpireTimed(secondOfNow) {
			//			log.Println("skip match room", check.Id)
			continue
		}
		if check.SubType&cpoint.SubType == 0 {
			//			log.Println("skip match room", check.Id)
			continue
		}
		distance := check.point - cpoint.point //两点之间的距离
		if distance < 0 {
			distance = -distance
		}
		matchRadius := nowRadius + check.CurrtRadius(secondOfNow) //邻居间产生交集最大距离
		//		log.Printf("r 玩家半径:%v,房间半径：%v,最大半径:%v,玩家中心点：%v,房间中心点：%v,两点直线距离：%v\n", nowRadius, check.CurrtRadius(secondOfNow), matchRadius, cpoint.point, check.point, distance)
		if distance <= matchRadius { //有交集#
			neighbor = append(neighbor, check)
			//			log.Printf("r 房间信息：%+v，玩家信息：%+v\n", check, cpoint)
			check.updaterIdx = i
			if needNum > 0 && len(neighbor) >= needNum {
				return
			}
		} else if matchRadius >= maxRadius { //超出了最大查找范围无需再向边缘查找
			break
		}
	}
	return
}

/**
根据节点查找所在列表中的下标(注意：该find节点必须在列表中存在，否则无法查找)
bool:是否查找到
int: bool=true 表示真实下标
      bool=false 表示可以插入的下标,int=[0,len]
*/
func (nd *LineNodeList) FindNode(find *LineNode) (bool, int) {
	//二分法找相同数值进行快速定位坐标范围
	i := sort.Search(nd.Len(), func(i int) bool { return nd.entries[i].point >= find.point })
	ti := i
	found := false
	if size := nd.Len(); i < size {
		if nd.entries[i] != find {
			//相同数值后再定位具体指针的节点
			for i = i + 1; i < size; i++ {
				if nd.entries[i] == find {
					found = true
				}
			}
		} else {
			found = true
		}
	}
	if found {
		return true, i
	} else {
		return false, ti
	}
}

/**
根据节点查找开始遍历的邻居节点下标
*/
func (nd *LineNodeList) SearchNeighbor(find *LineNode) int {
	//二分法找相同数值进行快速定位坐标范围
	i := sort.Search(nd.Len(), func(i int) bool { return nd.entries[i].point >= find.point })
	if size := nd.Len(); i < size {
		//相同数值后再定位具体指针的节点
		for i = i + 1; i < size; i++ {
			if nd.entries[i].point != find.point {
				i--
				break
			}
		}
	}
	return i
}

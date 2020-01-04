package match

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

var (
	_test_starttime int

	listplayer *LineNodeList
	listroom   *LineNodeList
)

func TestLineNodeList_Init(t *testing.T) {
	_test_starttime = time.Now().Second()
	listplayer = NewLineNodeList()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		start := 100
		for i := start; i < start+100000; i += 10 {
			listplayer.AddNode(NewLineNode(i, 0, 50, 15, _test_starttime))
		}
		wg.Done()
	}()

	listroom = NewLineNodeList()
	go func() {
		r := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
		start := 5000
		for i := start; i < start+50000; i += 20 {
			rangz := r.Intn(200)
			if rangz < 50 {
				rangz = 0
			}
			listroom.AddNode(NewLineNode(i, rangz, 0, 0, _test_starttime))
		}
		wg.Done()
	}()
	wg.Wait()
}

//测试添加节点 go test -v  -run TestLineNodeList_AddNode
func TestLineNodeList_AddNode(t *testing.T) {
	testaddnum := 5000
	t.Run("add_player", func(t *testing.T) {
		t.Logf("base player num:%d", listplayer.Len())
		r := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
		for i := 0; i < testaddnum; i++ {
			listplayer.AddNode(NewLineNode(r.Intn(5000), 0, 50, 15, _test_starttime))
		}
		t.Logf("now player num:%d", listplayer.Len())
	})
	t.Run("add_room", func(t *testing.T) {
		t.Logf("base room num:%d", listroom.Len())
		r := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
		for i := 0; i < testaddnum; i++ {
			rangz := r.Intn(200)
			if rangz < 100 {
				rangz = 0
			}
			listroom.AddNode(NewLineNode(r.Intn(5000), rangz, 0, 0, _test_starttime))
		}
		t.Logf("now room num:%d", listroom.Len())
	})
}

//测试选取匹配节点
func TestLineNodeList_SearchRange(t *testing.T) {
	r := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
	now := _test_starttime + r.Intn(15)
	t.Logf("base player num:%d", listplayer.Len())
	t.Logf("base room num:%d", listroom.Len())
	//所有未进入房间的玩家进行匹配房间
	glIndex1 := -1
	deleteplayer1 := 0
	delteroom1 := 0
	for glIndex1 < listplayer.Len()-1 {
		listplayer.Iterator(glIndex1+1, func(player *LineNode, crtIndex int) bool {
			glIndex1 = crtIndex
			if player.NeighborMax == 1 { //直接开房，不需要找房间
				return true
			}
			if player.ExpireTimed(now) { //玩家退出匹配了，外部将之ExpireTime设置成-1
				return true
			}
			//玩家找房间
			rooms := listroom.SearchRange(now, player, 1)
			if len(rooms) == 1 { //找到房间，把房间移除
				//type 1 直接将玩家从队列中删除
				//玩家加入房间，把玩家移除
				if index := listplayer.RemoveNode(player); index == -1 { //玩家删除失败跳过
					t.Errorf("remove selected room player,no found player %+v", player)
					return true
				} else { //玩家移除成功，从glIndex1再进行玩家筛选
					glIndex1--
					deleteplayer1++

					room := rooms[0]
					rangz := r.Intn(200)
					if rangz < 100 { //房间满了移除房间
						if index := listroom.RemoveNode(room); index == -1 { //房间删除失败跳过
							t.Errorf("remove full room node,no found room %+v", room)
						} else {
							delteroom1++
						}
					}
					return false
				}

				// type 2 将玩家设置为过期
				//				player.ExpireTime = -1
				//				deleteplayer1++
				//				return true
			} else {
				return true
			}
		})
	}
	t.Logf("match all room ,now player num:%d,delete num:%d;now room num:%d,delete num:%d", listplayer.Len(), deleteplayer1, listroom.Len(), delteroom1)
	//所有玩家之间进行匹配,创建新的房间
	glIndex2 := -1
	deleteplayer2 := 0
	addroom2 := 0
	for glIndex2 < listplayer.Len()-1 {
		listplayer.Iterator(glIndex2+1, func(player *LineNode, crtIndex int) bool {
			glIndex2 = crtIndex
			if player.ExpireTimed(now) { //玩家退出匹配了，外部将之ExpireTime设置成-1
				return true
			}
			//玩家找邻居
			neighbers := listplayer.SearchRange(now, player, r.Intn(10))
			rangz := r.Intn(200)
			if rangz < 50 || len(neighbers) > 0 { //人不足有一定几率自己创建房间

				//type 1
				if index := listplayer.RemoveNode(player); index == -1 { //玩家自己删除失败跳过
					t.Errorf("remove selected player,no found player %+v", player)
					return true
				} else { //玩家移除成功，从glIndex2再进行玩家筛选
					glIndex2--
					deleteplayer2++
				}

				//type 2
				//				player.ExpireTime = -1
				//				deleteplayer2++

				min := player.point
				max := player.point
				for _, neighber := range neighbers {
					if neighber.point < min {
						min = neighber.point
					} else if neighber.point > max {
						max = neighber.point
					}
				}
				listroom.AddNode(NewLineNode((min+max)>>1, (min+max)>>1-min, 0, 0, _test_starttime))
				addroom2++

				minIndex := glIndex2
				for _, neighber := range neighbers {
					//type 1
					//玩家加入房间，把玩家移除
					if index := listplayer.RemoveNode(neighber); index == -1 { //邻居删除失败跳过
						t.Errorf("remove selected player neighber,no found neighber %+v", neighber)
					} else { //玩家移除成功，从glIndex2再进行玩家筛选
						if index < minIndex {
							minIndex = index
						}
						deleteplayer2++
					}

					//type 2
					//					neighber.ExpireTime = -1
					//					deleteplayer2++

				}

				//type 1
				if minIndex != glIndex2 {
					glIndex2 = minIndex - 1
				}

				return false
			} else {
				return true
			}
		})
	}
	t.Logf("match all player ,now player num:%d,delete num:%d;now room num:%d,add num:%d", listplayer.Len(), deleteplayer2, listroom.Len(), addroom2)
}

//测试删除所有过期节点
func TestLineNodeList_CheckExpire(t *testing.T) {
	t.Run("expire_room", func(t *testing.T) {
		t.Logf("base room num:%d", listroom.Len())
		waite_remove := make(map[string]bool)
		r := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
		listroom.Iterator(0, func(node *LineNode, crtIndex int) bool { //先随机部分过期
			outtime := r.Intn(200)
			if outtime < 100 {
				node.ExpireTime = -1
			}
			if node.ExpireTimed(_test_starttime) {
				waite_remove[fmt.Sprintf("%p", node)] = true
			}
			return true
		})
		glIndex := -1
		for glIndex < listroom.Len()-1 {
			listroom.Iterator(glIndex+1, func(node *LineNode, crtIndex int) bool {
				glIndex = crtIndex
				if node.ExpireTimed(_test_starttime) {
					if index := listroom.RemoveNode(node); index == -1 {
						t.Errorf("remove expire room node,no found %p,index=%d,glIndex=%d", node, index, glIndex)
					} else {
						delete(waite_remove, fmt.Sprintf("%p", node))
						glIndex--
					}
					return false
				} else {
					return true
				}
			})
		}
		if len(waite_remove) != 0 {
			t.Errorf("expire node not remove over:%d,glIndex=%d,room=%d", len(waite_remove), glIndex, listroom.Len())
			t.Fail()
		}
		t.Logf("now room num:%d", listroom.Len())
	})

	t.Run("expire_player", func(t *testing.T) {
		t.Logf("base player num:%d", listplayer.Len())
		glIndex1 := -1
		for glIndex1 < listplayer.Len()-1 {
			listplayer.Iterator(glIndex1+1, func(node *LineNode, crtIndex int) bool {
				glIndex1 = crtIndex
				if node.ExpireTimed(_test_starttime) {
					if index := listplayer.RemoveNode(node); index == -1 {
						t.Errorf("remove expire player node,no found %p,index=%d,glIndex=%d", node, index, glIndex1)
					} else {
						glIndex1--
					}
					return false
				} else {
					return true
				}
			})
		}
		t.Logf("now player num:%d", listplayer.Len())
	})

}

//测试依次删除节点
func TestLineNodeList_CheckRemove(t *testing.T) {
	t.Run("remove_room", func(t *testing.T) {
		t.Logf("base room num:%d", listroom.Len())
		for listroom.Len() != 0 {
			listroom.Iterator(0, func(node *LineNode, crtIndex int) bool {
				if i := listroom.RemoveNode(node); i == -1 {
					t.Errorf("no found room node %p,point=%v", node, node.point)
				} else {
					//t.Logf("room remove %p,point=%v", node, node.point)
				}
				return false
			})
		}
		t.Logf("now room num:%d", listroom.Len())
	})

	t.Run("remove_player", func(t *testing.T) {
		t.Logf("base player num:%d", listplayer.Len())
		for listplayer.Len() != 0 {
			listplayer.Iterator(0, func(node *LineNode, crtIndex int) bool {
				if i := listplayer.RemoveNode(node); i == -1 {
					t.Errorf("no found player node %p,point=%v", node, node.point)
				} else {
					//t.Logf("player remove %p,point=%v", node, node.point)
				}
				return false
			})
		}
		t.Logf("now player num:%d", listplayer.Len())
	})
}

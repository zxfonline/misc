package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/zxfonline/misc/match"
)

var (
	matchData = []PlayerInfo{
		PlayerInfo{Id: 1, WaitTime: 0},
		PlayerInfo{Id: 2, WaitTime: 1},
		PlayerInfo{Id: 3, WaitTime: 2},
		PlayerInfo{Id: 4, WaitTime: 3},
		PlayerInfo{Id: 5, WaitTime: 4},
		PlayerInfo{Id: 6, WaitTime: 5},
		PlayerInfo{Id: 7, WaitTime: 6},
		PlayerInfo{Id: 8, WaitTime: 7},
		PlayerInfo{Id: 9, WaitTime: 8},
		PlayerInfo{Id: 10, WaitTime: 9},
		PlayerInfo{Id: 11, WaitTime: 10},
		PlayerInfo{Id: 12, WaitTime: 11},
	}
)

var (
	_test_starttime = time.Now().Second()
	listPlayer      = match.NewLineNodeList()
)

type PlayerInfo struct {
	Id       int
	WaitTime int
}

//匹配完成的队伍信息
type MathRoomInfo struct {
	Players []*PlayerInfo //加入的玩家
}

func main() {
	log.Println("匹配信息:")
	if matches := SearchRange(_test_starttime); len(matches) > 0 {
		matchcalc(matches)
	}
	log.Println("未匹配的玩家:")
	listPlayer.Iterator(0, func(player *match.LineNode, crtIndex int) bool {
		log.Printf(">>>玩家ID:%d,等待时间:%d\n", player.Id, _test_starttime-player.InsertTime())
		return true
	})
}

func matchcalc(matches []*MathRoomInfo) {
	for _, match := range matches {
		log.Println("房间ID")
		for _, user := range match.Players {
			log.Println(fmt.Sprintf(">>>成员ID:%d,等待时间:%d", user.Id, user.WaitTime))
		}
	}
}

type PlayerMatchCfg struct {
	// 匹配的最多人数，在等待时间内一旦满了就开房
	MatchNumMax int `json:"match_num_max"`
	//匹配的最少人数，即使等待时间到了，人数不够最少也不能开房
	MatchNumMin int `json:"match_num_min"`
	// 匹配等待最大时间，达到该时间后匹配够最少人数值即可开房 单位:秒
	WaitTimeLimit int `json:"wait_time_limit"`
}

type FightRoomMatchCfg struct {
	PlayLockTime    int64 `json:"play_lock_time"`    // 房间内游戏开始多少分钟锁定，不允许玩家进入并用AI补齐剩余玩家数量单位:秒
	MaxPlayerNumber int32 `json:"max_player_number"` // 房间一局游戏最大玩家数量，超过则新开游戏
}

var (
	cfg = PlayerMatchCfg{
		MatchNumMax:   5,
		MatchNumMin:   1,
		WaitTimeLimit: 8,
	}
)

func init() {
	//logFile, err := fileutil.OpenFile("./log/玩家战斗匹配信息.log", fileutil.DefaultFileFlag, fileutil.DefaultFileMode)
	//if err != nil {
	//	log.Fatalln("open file error !")
	//	os.Exit(-1)
	//	return
	//}
	log.SetPrefix("")
	//log.SetOutput(logFile)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for _, v := range matchData {
			node := match.NewLineNode(0, 5, 0, cfg.WaitTimeLimit, _test_starttime-v.WaitTime)
			node.Id = v.Id
			node.NeighborMax = cfg.MatchNumMax
			node.NeighborMin = cfg.MatchNumMin
			listPlayer.AddNode(node)
		}
		wg.Done()
	}()

	wg.Wait()
}

//测试选取匹配节点
func SearchRange(secondOfNow int) (matches []*MathRoomInfo) {
	log.Println(fmt.Sprintf("base player num:%d", listPlayer.Len()))
	//所有玩家之间进行匹配,创建新的房间
	glIndex := -1
	deletePlayer := 0
	createdRoom := 0
	for glIndex < listPlayer.Len()-1 {
		listPlayer.Iterator(glIndex+1, func(player *match.LineNode, crtIndex int) bool {
			glIndex = crtIndex
			if player.ExpireTimed(secondOfNow) { //玩家退出匹配了，外部将之ExpireTime设置成-1
				return true
			}
			//玩家找邻居
			neighbours := listPlayer.SearchRange(secondOfNow, player, player.NeighborMax-1)
			if len(neighbours) >= player.NeighborMax-1 || (player.WaitFull(secondOfNow) && len(neighbours) >= player.NeighborMin-1) { //找够了,开房 or 等待的时间够了,满足了最少人数,开房
				infos := make([]*PlayerInfo, 0, len(neighbours)+1)
				if index := listPlayer.RemoveNode(player); index == -1 { //玩家自己删除失败跳过
					log.Println(fmt.Sprintf("remove selected player,no found player %+v", player))
					return true
				} else { //玩家移除成功，从glIndex2再进行玩家筛选
					glIndex--
					deletePlayer++
					infos = append(infos, &PlayerInfo{Id: player.Id, WaitTime: secondOfNow - player.InsertTime()})
				}
				createdRoom++

				minIndex := glIndex
				for _, neighbor := range neighbours {
					//玩家加入房间，把玩家移除
					if index := listPlayer.RemoveNode(neighbor); index == -1 { //邻居删除失败跳过
						log.Println(fmt.Sprintf("remove selected player neighbor,no found neighbor %+v", neighbor))
					} else { //玩家移除成功，从glIndex2再进行玩家筛选
						if index < minIndex {
							minIndex = index
						}
						deletePlayer++

						infos = append(infos, &PlayerInfo{Id: neighbor.Id, WaitTime: secondOfNow - neighbor.InsertTime()})
					}
				}

				if minIndex != glIndex {
					glIndex = minIndex - 1
				}
				matches = append(matches, &MathRoomInfo{
					Players: infos,
				})
				return false
			} else {
				return true
			}
		})
	}
	log.Println(fmt.Sprintf("match all player ,secondOfNow player num:%d,delete num:%d;add num:%d", listPlayer.Len(), deletePlayer, createdRoom))
	return
}

package model

import (
	"strings"

	"github.com/go-xorm/xorm"
)

// Video ...
type Video struct {
	Model        `xorm:"extends" json:"-"`
	FindNo       string   `json:"-"`                           //查找号
	Bangumi      string   `xorm:"bangumi"`                     //番組
	ThumbHash    string   `xorm:"thumb_hash"`                  //缩略图
	Intro        string   `xorm:"varchar(2048)" json:"intro"`  //简介
	Alias        []string `xorm:"json" json:"alias"`           //别名，片名
	SourceHash   string   `xorm:"source_hash"`                 //原片地址
	Key          string   `json:"-"`                           //秘钥
	M3U8         string   `xorm:"m3u8" json:"-"`               //M3U8名
	M3U8Hash     string   `xorm:"m3u8_hash" json:"m3u8_hash"`  //切片地址
	PosterHash   string   `xorm:"poster_hash"`                 //海报地址
	Role         []string `xorm:"json" json:"role"`            //主演
	Director     string   `json:"-"`                           //导演
	Systematics  string   `json:"-"`                           //分级
	Season       string   `json:"-"`                           //季
	TotalEpisode string   `json:"-"`                           //总集数
	Episode      string   `json:"-"`                           //集数
	Producer     string   `json:"-"`                           //生产商
	Publisher    string   `json:"-"`                           //发行商
	Type         string   `json:"-"`                           //类型：film，FanDrama
	Format       string   `json:"format"`                      //输出格式：3D，2D,VR(VR格式：Half-SBS：左右半宽,Half-OU：上下半高,SBS：左右全宽)
	Language     string   `json:"-"`                           //语言
	Caption      string   `json:"-"`                           //字幕
	Group        string   `json:"-"`                           //分组
	Index        string   `json:"-"`                           //索引
	Date         string   `json:"-"`                           //发行日期
	Sharpness    string   `json:"sharpness"`                   //清晰度
	Visit        uint64   `json:"-" xorm:"notnull default(0)"` //访问数
	Series       string   `json:"series"`                      //系列
	Tags         []string `xorm:"json" json:"tags"`            //标签
	Length       string   `json:"length"`                      //时长
	MagnetLinks  []string `json:"-"`                           //磁链
	Uncensored   bool     `json:"uncensored"`                  //有码,无码
}

// GetID ...
func (v *Video) GetID() string {
	return v.ID
}

// SetID ...
func (v *Video) SetID(s string) {
	v.ID = s
}

// GetVersion ...
func (v *Video) GetVersion() int {
	return v.Version
}

// SetVersion ...
func (v *Video) SetVersion(i int) {
	v.Version = i
}

func init() {
	RegisterTable(Video{})
}

// FindVideo ...
func FindVideo(session *xorm.Session, ban string) (video *Video, e error) {
	video = new(Video)
	ban = strings.ReplaceAll(ban, "-", "")
	ban = strings.ReplaceAll(ban, "_", "")
	ban = strings.ToUpper(ban)
	_, e = DB().Where("find_no = ?", ban).Get(video)
	return
}

// Top ...
func Top(video *Video) (b bool, e error) {
	return DB().OrderBy("created_at desc").Get(video)
}

// TopList ...
func TopList(session *xorm.Session, limit int, start ...int) (videos *[]*Video, e error) {
	videos = new([]*Video)
	session = MustSession(session)
	if limit > 0 {
		session = session.Limit(limit, start...)
	}
	if e = session.OrderBy("visit desc").Find(videos); e != nil {
		return
	}
	return videos, nil
}

// AllVideos ...
func AllVideos(session *xorm.Session, limit int, start ...int) (videos *[]*Video, e error) {
	videos = new([]*Video)
	session = MustSession(session)
	if limit > 0 {
		session = session.Limit(limit, start...)
	}
	if e = session.Find(videos); e != nil {
		return
	}
	return videos, nil
}

// DeepFind ...
func DeepFind(s string, videos *[]*Video) (e error) {
	s1 := strings.ReplaceAll(s, "-", "")
	s1 = strings.ReplaceAll(s1, "_", "")
	s1 = strings.ToUpper(s1)
	e = DB().Where("find_no = ?", s1).OrderBy("season,episode asc").Find(videos)
	if e != nil || len(*videos) <= 0 {
		like := "%" + strings.ToUpper(s) + "%"
		e = DB().Where("find_no like ? ", like).
			Or("intro like ?", like).
			Or("role like ?", like).
			OrderBy("season,episode asc").
			Find(videos)
	}
	return e
}

func parseStr(s *string, d string) {
	if *s == "" {
		*s = d
	}
}

// AddOrUpdateVideo ...
func AddOrUpdateVideo(video *Video) (e error) {
	var tmp Video
	var found bool
	if video.ID != "" {
		found, e = DB().ID(video.ID).Get(&tmp)
	} else {
		//if video.Sharpness != "" {
		//	found, e = DB().Where("bangumi = ?", video.Bangumi).
		//		//TODO:which was only one?
		//		Where("season = ?", video.Season).
		//		Where("episode = ?", video.Episode).
		//		Where("sharpness = ?", video.Sharpness).
		//		Get(&tmp)
		//}else{
		found, e = DB().Where("bangumi = ?", video.Bangumi).
			//TODO:which was only one?
			Where("season = ?", video.Season).
			Where("episode = ?", video.Episode).
			//Where("sharpness = ?", video.Sharpness).
			Get(&tmp)
		//}
	}
	if e != nil {
		return e
	}

	if found {
		video.Version = tmp.Version
		video.ID = tmp.ID
		parseStr(&video.M3U8Hash, tmp.M3U8Hash)
		parseStr(&video.SourceHash, tmp.SourceHash)
		parseStr(&video.PosterHash, tmp.PosterHash)
		parseStr(&video.ThumbHash, tmp.ThumbHash)
		i, e := DB().ID(video.ID).Update(video)
		log.Infof("updated(%d): %+v", i, tmp)
		return e
	}
	_, e = DB().InsertOne(video)
	return
}

// Visited ...
func Visited(video *Video) (err error) {
	video.Visit++
	if _, err := DB().Where("bangumi = ?", video.Bangumi).Cols("visit").Update(video); err != nil {
		return err
	}
	return nil
}

// Clone ...
func (v *Video) Clone() (n *Video) {
	n = new(Video)
	*n = *v
	n.ID = ""
	return
}

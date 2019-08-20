package seed

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/glvd/seed/model"
	"github.com/go-xorm/xorm"
	"go.uber.org/atomic"
	"golang.org/x/xerrors"
)

// InfoType ...
type InfoType string

// InfoTypeNone ...
const InfoTypeNone InfoType = "none"

// InfoTypeJSON ...
const InfoTypeJSON InfoType = "json"

// InfoTypeBSON ...
const InfoTypeBSON InfoType = "bson"

// Information ...
type Information struct {
	*Seed
	infoType     InfoType
	Path         string
	ResourcePath string
	ProcList     []string
	Start        int
	video        chan *model.Video
}

// Option ...
func (info *Information) Option(seed *Seed) {
	informationOption(info)(seed)
}

// NewInformation ...
func NewInformation() *Information {
	info := new(Information)
	info.video = make(chan *model.Video, 3)
	return info
}

// OutputInfomation ...
func (info *Information) OutputInfomation() <-chan *model.Video {
	return info.video
}

// BeforeRun ...
func (info *Information) BeforeRun(seed *Seed) {
	info.Seed = seed
}

// AfterRun ...
func (info *Information) AfterRun(seed *Seed) {
}

func fixBson(s []byte) []byte {
	reg := regexp.MustCompile(`("_id")[ ]*[:][ ]*(ObjectId\(")[\w]{24}("\))[ ]*(,)[ ]*`)
	return reg.ReplaceAll(s, []byte(" "))
}

func video(source *VideoSource) (video *model.Video) {

	//always not null
	alias := *new([]string)
	aliasS := ""
	if source.Alias != nil && len(source.Alias) > 0 {
		alias = source.Alias
		aliasS = alias[0]
	}
	//always not null
	role := *new([]string)
	roleS := ""
	if source.Role != nil && len(source.Role) > 0 {
		role = source.Role
		roleS = role[0]
	}

	intro := source.Intro
	if intro == "" {
		intro = aliasS + " " + roleS
	}

	return &model.Video{
		FindNo:       strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(source.Bangumi, "-", ""), "_", "")),
		Bangumi:      strings.ToUpper(source.Bangumi),
		Type:         source.Type,
		Systematics:  source.Systematics,
		Sharpness:    source.Sharpness,
		Producer:     source.Producer,
		Language:     source.Language,
		Caption:      source.Caption,
		Intro:        intro,
		Alias:        alias,
		Role:         role,
		Director:     source.Director,
		Series:       source.Series,
		Tags:         source.Tags,
		Date:         source.Date,
		SourceHash:   source.SourceHash,
		Season:       MustString(source.Season, "1"),
		Episode:      MustString(source.Episode, "1"),
		TotalEpisode: MustString(source.TotalEpisode, "1"),
		Format:       MustString(source.Format, "2D"),
		Publisher:    source.Publisher,
		Length:       source.Length,
		MagnetLinks:  source.MagnetLinks,
		Uncensored:   source.Uncensored,
	}
}

// defaultUnfinished ...
func defaultUnfinished(name string) *model.Unfinished {
	_, file := filepath.Split(name)

	uncat := &model.Unfinished{
		Model:       model.Model{},
		Checksum:    "",
		Type:        "other",
		Relate:      "",
		Name:        file,
		Hash:        "",
		Sharpness:   "",
		Caption:     "",
		Encrypt:     false,
		Key:         "",
		M3U8:        "media.m3u8",
		SegmentFile: "media-%05d.ts",
		Sync:        false,
		Object:      new(model.VideoObject),
	}
	log.With("file", name).Info("calculate checksum")
	uncat.Checksum = model.Checksum(name)
	return uncat
}

func checkFileNotExist(path string) bool {
	_, e := os.Stat(path)
	if e != nil {
		if os.IsNotExist(e) {
			return true
		}
		return false
	}
	return false
}

// Run ...
func (info *Information) Run(ctx context.Context) {
	log.Info("Information running")
	var vs []*VideoSource
	isDefault := true
	switch info.infoType {
	case InfoTypeBSON:
		isDefault = false
		b, e := ioutil.ReadFile(info.Path)
		if e != nil {
			panic(e)
		}
		fixed := fixBson(b)
		reader := bytes.NewBuffer(fixed)
		e = LoadFrom(&vs, reader)
		if e != nil {
			panic(e)
		}
	case InfoTypeJSON:
		isDefault = false
		b, e := ioutil.ReadFile(info.Path)
		if e != nil {
			panic(e)
		}
		reader := bytes.NewBuffer(b)
		e = LoadFrom(&vs, reader)
		if e != nil {
			panic(e)
		}
	default:
	}
	if !isDefault {
		if vs == nil {
			log.Info("nil video source")
			return
		}
		log.With("filter", info.ProcList).Info("filter list")
		vs = filterProcList(vs, info.ProcList)
		failedSkip := atomic.NewBool(false)
		maxLimit := len(vs)
		m := make(map[string]string)
		for i := info.Start; i < maxLimit; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				source := vs[i]
				v := video(source)
				if !failedSkip.Load() {
					if source.Poster != "" {
						v.PosterHash = source.Poster
					} else {
						if source.PosterPath != "" {
							source.PosterPath = filepath.Join(info.ResourcePath, source.PosterPath)
							if checkFileNotExist(source.PosterPath) {
								log.With("index", i, "bangumi", source.Bangumi).Info("poster not found")
							} else {
								poster, e := addPosterHash(info.Database, source, "hash")
								if e != nil {
									log.Error(e)
									failedSkip.Store(true)
								} else {
									v.PosterHash = poster.Hash
									m[source.PosterPath] = poster.Hash
								}
							}
						}
					}

					if source.Thumb != "" {
						source.Thumb = filepath.Join(info.ResourcePath, source.Thumb)
						if checkFileNotExist(source.Thumb) {
							log.With("index", i, "bangumi", source.Bangumi).Info("thumb not found")
						} else {
							thumb, e := addThumbHash(info.Database, source, "hash")
							if e != nil {
								log.Error(e)
								failedSkip.Store(true)
							} else {
								v.ThumbHash = thumb.Hash
								m[source.Thumb] = thumb.Hash
							}
						}
					}
				}

				info.Database.PushCallback(func(database *Database, eng *xorm.Engine) (e error) {
					return model.AddOrUpdateVideo(eng.NewSession(), v)
				})
			}
		}
		log.Info("info end")
	}

	return
}

func addThumbHash(db *Database, source *VideoSource, hash string) (unf *model.Unfinished, e error) {
	unfinThumb := defaultUnfinished(source.Thumb)
	unfinThumb.Type = model.TypeThumb
	unfinThumb.Relate = source.Bangumi
	if source.Thumb != "" {
		unfinThumb.Hash = hash
		db.PushCallback(func(database *Database, eng *xorm.Engine) (e error) {
			return model.AddOrUpdateUnfinished(eng.NewSession(), unfinThumb)
		})
		return unfinThumb, nil
	}

	return nil, xerrors.New("no thumb")
}

func addPosterHash(db *Database, source *VideoSource, hash string) (unf *model.Unfinished, e error) {
	unfinPoster := defaultUnfinished(source.PosterPath)
	unfinPoster.Type = model.TypePoster
	unfinPoster.Relate = source.Bangumi

	if source.PosterPath != "" {
		unfinPoster.Hash = hash
		db.PushCallback(func(database *Database, eng *xorm.Engine) (e error) {
			return model.AddOrUpdateUnfinished(eng.NewSession(), unfinPoster)
		})
		return unfinPoster, nil
	}
	return nil, xerrors.New("no poster")
}

func addVideo(db *Database, video *model.Video) {

}

func filterProcList(sources []*VideoSource, filterList []string) (vs []*VideoSource) {
	if filterList == nil || len(filterList) == 0 {
		return sources
	}
	for _, source := range sources {
		for _, v := range filterList {
			if source.Bangumi == v {
				vs = append(vs, source)
			}
		}
	}
	return
}

// VideoSource ...
type VideoSource struct {
	Bangumi      string    `json:"bangumi"`       //番号 no
	SourceHash   string    `json:"source_hash"`   //原片hash
	Type         string    `json:"type"`          //类型：film，FanDrama
	Format       string    `json:"format"`        //输出：3D，2D
	VR           string    `json:"vr"`            //VR格式：左右，上下，平面
	Thumb        string    `json:"thumb"`         //缩略图
	Intro        string    `json:"intro"`         //简介 title
	Alias        []string  `json:"alias"`         //别名，片名
	VideoEncode  string    `json:"video_encode"`  //视频编码
	AudioEncode  string    `json:"audio_encode"`  //音频编码
	Files        []string  `json:"files"`         //存放路径
	HashFiles    []string  `json:"hash_files"`    //已上传Hash
	CheckFiles   []string  `json:"check_files"`   //Unfinished checksum
	Slice        bool      `json:"sliceAdd"`      //是否HLS切片
	Encrypt      bool      `json:"encrypt"`       //加密
	Key          string    `json:"key"`           //秘钥
	M3U8         string    `json:"m3u8"`          //M3U8名
	SegmentFile  string    `json:"segment_file"`  //ts切片名
	PosterPath   string    `json:"poster_path"`   //海报路径
	Poster       string    `json:"poster"`        //海报HASH
	ExtendList   []*Extend `json:"extend_list"`   //扩展信息
	Role         []string  `json:"role"`          //角色列表 stars
	Director     string    `json:"director"`      //导演
	Systematics  string    `json:"systematics"`   //分级
	Season       string    `json:"season"`        //季
	Episode      string    `json:"episode"`       //集数
	TotalEpisode string    `json:"total_episode"` //总集数
	Sharpness    string    `json:"sharpness"`     //清晰度
	Publish      string    `json:"publish"`       //发行日
	Date         string    `json:"date"`          //发行日
	Length       string    `json:"length"`        //片长
	Producer     string    `json:"producer"`      //制片商
	Series       string    `json:"series"`        //系列
	Tags         []string  `json:"tags"`          //标签
	Publisher    string    `json:"publisher"`     //发行商
	Language     string    `json:"language"`      //语言
	Caption      string    `json:"caption"`       //字幕
	MagnetLinks  []string  `json:"magnet_links"`  //磁链
	Uncensored   bool      `json:"uncensored"`    //有码,无码
}

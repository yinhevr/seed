package seed

import (
	"bufio"
	"context"
	"encoding/json"
	"github.com/go-xorm/xorm"
	shell "github.com/godcong/go-ipfs-restapi"
	jsoniter "github.com/json-iterator/go"
	"github.com/yinhevr/seed/model"
	"github.com/yinhevr/seed/old"
	"golang.org/x/xerrors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// TransferStatus ...
type TransferStatus string

// TransferFlagNone ...
const (
	TransferStatusNone      TransferStatus = "none"
	TransferFlagVerify      TransferStatus = "verify"
	TransferStatusToJSON    TransferStatus = "json"
	TransferStatusFromOther TransferStatus = "other"
	TransferStatusFromOld   TransferStatus = "old"
	TransferStatusUpdate    TransferStatus = "update"
	TransferStatusDelete    TransferStatus = "delete"
)

// transfer ...
type transfer struct {
	shell      *shell.Shell
	unfinished map[string]*model.Unfinished
	videos     map[string]*model.Video
	workspace  string
	flag       InfoFlag
	status     TransferStatus
	path       string
	reader     io.Reader
}

// BeforeRun ...
func (transfer *transfer) BeforeRun(seed *Seed) {
	transfer.shell = seed.Shell
	transfer.workspace = seed.Workspace
	transfer.unfinished = seed.Unfinished
	transfer.videos = seed.Videos

}

// AfterRun ...
func (transfer *transfer) AfterRun(seed *Seed) {
	seed.Videos = transfer.videos
	seed.Unfinished = transfer.unfinished
}

// TransferOption ...
func TransferOption(t *transfer) Options {
	return func(seed *Seed) {
		seed.thread[StepperTransfer] = t
	}
}

// Transfer ...
func Transfer(path string, from InfoFlag, status TransferStatus) Options {
	t := &transfer{
		path:   path,
		flag:   from,
		status: status,
	}
	return TransferOption(t)
}

func addThumbHash(shell *shell.Shell, source *VideoSource) (*model.Unfinished, error) {
	unfinThumb := defaultUnfinished(source.Thumb)
	unfinThumb.Type = model.TypeThumb
	unfinThumb.Relate = source.Bangumi
	if source.Thumb != "" {
		abs, e := filepath.Abs(source.Thumb)
		if e != nil {
			return nil, e
		}

		object, e := shell.AddFile(abs)
		if e != nil {
			return nil, e
		}

		unfinThumb.Hash = object.Hash
		e = model.AddOrUpdateUnfinished(unfinThumb)
		if e != nil {
			return nil, e
		}
		return unfinThumb, nil
	}

	return nil, xerrors.New("no thumb")
}

func addPosterHash(shell *shell.Shell, source *VideoSource) (*model.Unfinished, error) {
	unfinPoster := defaultUnfinished(source.PosterPath)
	unfinPoster.Type = model.TypePoster
	unfinPoster.Relate = source.Bangumi

	if source.PosterPath != "" {
		abs, e := filepath.Abs(source.PosterPath)
		if e != nil {
			return nil, e
		}
		object, e := shell.AddFile(abs)
		if e != nil {
			return nil, e
		}
		unfinPoster.Hash = object.Hash
		e = model.AddOrUpdateUnfinished(unfinPoster)
		if e != nil {
			return nil, e
		}
		return unfinPoster, nil
	}
	return nil, xerrors.New("no poster")
}

func insertOldToUnfinished(ban string, obj *old.Object) error {
	hash := ""
	if obj.Link != nil {
		hash = obj.Link.Hash
	}
	unf := &model.Unfinished{
		Checksum:    hash,
		Type:        model.TypeSlice,
		Relate:      ban,
		Name:        ban,
		Hash:        hash,
		Sharpness:   "",
		Caption:     "",
		Encrypt:     false,
		Key:         "",
		M3U8:        "",
		SegmentFile: "",
		Sync:        false,
		Object:      ObjectFromOld(obj),
	}
	return model.AddOrUpdateUnfinished(unf)

}

// ObjectFromOld ...
func ObjectFromOld(obj *old.Object) *model.VideoObject {
	return &model.VideoObject{
		Links: obj.Links,
		Link:  obj.Link,
	}
}

func transferFromOld(engine *xorm.Engine) (e error) {
	objects := old.LoadOld(engine)
	log.With("size", len(objects)).Info("objects")
	for ban, obj := range objects {
		e := insertOldToUnfinished(ban, obj)
		if e != nil {
			log.With("bangumi", ban).Error(e)
			continue
		}
		vd, e := model.FindVideo(nil, ban)
		if e != nil || vd.ID == "" {
			log.With("bangumi", ban).Error(e)
			continue
		}
		log.With("bangumi", ban, "video", vd).Info("video update")
		if strings.TrimSpace(vd.M3U8Hash) == "" && obj.Link != nil {
			log.With("hash:", obj.Link.Hash, "bangumi", ban).Info("info")
			vd.M3U8Hash = obj.Link.Hash
			e = model.AddOrUpdateVideo(vd)
			if e != nil {
				log.With("bangumi", ban).Error(e)
				continue
			}
		} else {

		}

	}
	return e
}

func transferFromOther(engine *xorm.Engine) (e error) {
	fromList := new([]*model.Video)
	e = engine.Find(fromList)
	if e != nil {
		return
	}
	for _, from := range *fromList {
		video, e := model.FindVideo(nil, from.Bangumi)
		if e != nil {
			log.Error(e)
			continue
		}
		if video.ID == "" {
			continue
		}
		video.M3U8Hash = MustString(from.M3U8Hash, video.M3U8Hash)
		video.Sharpness = MustString(from.Sharpness, video.Sharpness)
		video.SourceHash = MustString(from.SourceHash, video.SourceHash)
		e = model.AddOrUpdateVideo(video)
		if e != nil {
			log.With("bangumi", from.Bangumi).Error(e)
			continue
		}
	}
	return e
}

func transferUpdate(engine *xorm.Engine) (e error) {
	fromList := new([]*model.Unfinished)
	e = engine.Find(fromList)
	if e != nil {
		return
	}
	for _, from := range *fromList {
		video, e := model.FindVideo(model.DB().Where("episode = ?", NumberIndex(from.Relate)), onlyName(from.Relate))
		if e != nil {
			log.Error(e)
			continue
		}

		if from.Type == model.TypeSlice {
			video.Sharpness = MustString(from.Sharpness, video.Sharpness)
			video.M3U8Hash = MustString(from.Hash, video.M3U8Hash)
		} else if from.Type == model.TypeVideo {
			video.Sharpness = MustString(from.Sharpness, video.Sharpness)
			video.SourceHash = MustString(from.Hash, video.SourceHash)
		} else {

		}
		e = model.AddOrUpdateVideo(video)
		if e != nil {
			log.With("bangumi", video.Bangumi, "index", video.Episode).Error(e)
			continue
		}
	}
	return e
}

func transferToJSON(to string) (e error) {
	videos, e := model.AllVideos(model.DB().Where("m3u8_hash <> ?", ""), 0)
	if e != nil {
		return e
	}
	bytes, e := jsoniter.Marshal(videos)
	if e != nil {
		return e
	}
	file, e := os.OpenFile(to, os.O_CREATE|os.O_SYNC|os.O_RDWR|os.O_APPEND, os.ModePerm)
	if e != nil {
		return e
	}
	defer file.Close()
	n, e := file.Write(bytes)
	log.Infof("write(%d)", n)
	return e
}

// Run ...
func (transfer *transfer) Run(ctx context.Context) {
	if transfer.flag == InfoFlagSQLite {
		fromDB, e := model.InitDB("sqlite3", transfer.path)
		if e != nil {
			log.Error(e)
			return
		}
		e = fromDB.Sync2(model.Video{})
		if e != nil {
			log.Error(e)
			return
		}
		switch transfer.status {
		case TransferStatusFromOld:
			if err := transferFromOld(fromDB); err != nil {
				log.Error(err)
				return
			}
		//update flag video flag other sqlite3
		case TransferStatusFromOther:
			if err := transferFromOther(fromDB); err != nil {
				return
			}
		//update flag unfinished flag other sqlite3
		case TransferStatusUpdate:
			if err := transferUpdate(fromDB); err != nil {
				return
			}
		case TransferStatusToJSON:
			if err := transferToJSON(transfer.path); err != nil {
				return
			}
		}
	}

}

// LoadFrom ...
func LoadFrom(vs *[]*VideoSource, reader io.Reader) (e error) {
	dec := json.NewDecoder(bufio.NewReader(reader))
	return dec.Decode(vs)
}

// TransferTo ...
func TransferTo(eng *xorm.Engine, limit int) (e error) {
	i, e := model.DB().Count(&model.Video{})
	if e != nil || i <= 0 {
		return e
	}
	if limit == 0 {
		limit = 10
	}
	for x := 0; x <= int(i); x += limit {
		var videos []*model.Video
		if e = model.DB().Limit(limit, x).Find(&videos); e != nil {
			return xerrors.Errorf("transfer error with:%d,%+v", x, e)
		}
		for _, v := range videos {
			log.Info("get:", v.Bangumi)
		}
		insert, e := eng.Insert(videos)
		log.Info(insert, e)
	}

	return nil
}

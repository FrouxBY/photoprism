package api

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/photoprism/photoprism/internal/acl"
	"github.com/photoprism/photoprism/internal/entity"
	"github.com/photoprism/photoprism/internal/event"
	"github.com/photoprism/photoprism/internal/form"
	"github.com/photoprism/photoprism/internal/i18n"
	"github.com/photoprism/photoprism/internal/photoprism"
	"github.com/photoprism/photoprism/internal/query"
	"github.com/photoprism/photoprism/internal/service"
	"github.com/photoprism/photoprism/internal/thumb"
	"github.com/photoprism/photoprism/pkg/fs"
	"github.com/photoprism/photoprism/pkg/rnd"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/photoprism/photoprism/pkg/txt"
)

// GET /api/v1/albums
func GetAlbums(router *gin.RouterGroup) {
	router.GET("/albums", func(c *gin.Context) {
		s := Auth(SessionID(c), acl.ResourceAlbums, acl.ActionSearch)

		if s.Invalid() {
			AbortUnauthorized(c)
			return
		}

		var f form.AlbumSearch

		err := c.MustBindWith(&f, binding.Form)

		if err != nil {
			AbortBadRequest(c)
			return
		}

		// Guest permissions are limited to shared albums.
		if s.Guest() {
			f.ID = s.Shares.String()
		}

		result, err := query.AlbumSearch(f)

		if err != nil {
			c.AbortWithStatusJSON(400, gin.H{"error": txt.UcFirst(err.Error())})
			return
		}

		c.Header("X-Count", strconv.Itoa(len(result)))
		c.Header("X-Limit", strconv.Itoa(f.Count))
		c.Header("X-Offset", strconv.Itoa(f.Offset))

		c.JSON(http.StatusOK, result)
	})
}

// GET /api/v1/albums/:uid
func GetAlbum(router *gin.RouterGroup) {
	router.GET("/albums/:uid", func(c *gin.Context) {
		s := Auth(SessionID(c), acl.ResourceAlbums, acl.ActionRead)

		if s.Invalid() {
			AbortUnauthorized(c)
			return
		}

		id := c.Param("uid")
		m, err := query.AlbumByUID(id)

		if err != nil {
			Abort(c, http.StatusNotFound, i18n.ErrAlbumNotFound)
			return
		}

		c.JSON(http.StatusOK, m)
	})
}

// POST /api/v1/albums
func CreateAlbum(router *gin.RouterGroup) {
	router.POST("/albums", func(c *gin.Context) {
		s := Auth(SessionID(c), acl.ResourceAlbums, acl.ActionCreate)

		if s.Invalid() {
			AbortUnauthorized(c)
			return
		}

		var f form.Album

		if err := c.BindJSON(&f); err != nil {
			AbortBadRequest(c)
			return
		}

		m := entity.NewAlbum(f.AlbumTitle, entity.AlbumDefault)
		m.AlbumFavorite = f.AlbumFavorite

		log.Debugf("album: creating %+v %+v", f, m)

		if res := entity.Db().Create(m); res.Error != nil {
			AbortAlreadyExists(c, txt.Quote(m.AlbumTitle))
			return
		}

		event.SuccessMsg(i18n.MsgAlbumCreated)

		UpdateClientConfig()

		PublishAlbumEvent(EntityCreated, m.AlbumUID, c)

		c.JSON(http.StatusOK, m)
	})
}

// PUT /api/v1/albums/:uid
func UpdateAlbum(router *gin.RouterGroup) {
	router.PUT("/albums/:uid", func(c *gin.Context) {
		s := Auth(SessionID(c), acl.ResourceAlbums, acl.ActionUpdate)

		if s.Invalid() {
			AbortUnauthorized(c)
			return
		}

		uid := c.Param("uid")
		m, err := query.AlbumByUID(uid)

		if err != nil {
			Abort(c, http.StatusNotFound, i18n.ErrAlbumNotFound)
			return
		}

		f, err := form.NewAlbum(m)

		if err != nil {
			log.Error(err)
			AbortSaveFailed(c)
			return
		}

		if err := c.BindJSON(&f); err != nil {
			log.Error(err)
			AbortBadRequest(c)
			return
		}

		if err := m.SaveForm(f); err != nil {
			log.Error(err)
			AbortSaveFailed(c)
			return
		}

		UpdateClientConfig()

		event.SuccessMsg(i18n.MsgAlbumSaved)

		PublishAlbumEvent(EntityUpdated, uid, c)

		c.JSON(http.StatusOK, m)
	})
}

// DELETE /api/v1/albums/:uid
func DeleteAlbum(router *gin.RouterGroup) {
	router.DELETE("/albums/:uid", func(c *gin.Context) {
		s := Auth(SessionID(c), acl.ResourceAlbums, acl.ActionDelete)

		if s.Invalid() {
			AbortUnauthorized(c)
			return
		}

		conf := service.Config()
		id := c.Param("uid")

		m, err := query.AlbumByUID(id)

		if err != nil {
			Abort(c, http.StatusNotFound, i18n.ErrAlbumNotFound)
			return
		}

		PublishAlbumEvent(EntityDeleted, id, c)

		conf.Db().Delete(&m)

		UpdateClientConfig()

		event.SuccessMsg(i18n.MsgAlbumDeleted, txt.Quote(m.AlbumTitle))

		c.JSON(http.StatusOK, m)
	})
}

// POST /api/v1/albums/:uid/like
//
// Parameters:
//   uid: string Album UID
func LikeAlbum(router *gin.RouterGroup) {
	router.POST("/albums/:uid/like", func(c *gin.Context) {
		s := Auth(SessionID(c), acl.ResourceAlbums, acl.ActionLike)

		if s.Invalid() {
			AbortUnauthorized(c)
			return
		}

		conf := service.Config()
		id := c.Param("uid")
		album, err := query.AlbumByUID(id)

		if err != nil {
			Abort(c, http.StatusNotFound, i18n.ErrAlbumNotFound)
			return
		}

		album.AlbumFavorite = true
		conf.Db().Save(&album)

		UpdateClientConfig()
		PublishAlbumEvent(EntityUpdated, id, c)

		c.JSON(http.StatusOK, i18n.NewResponse(http.StatusOK, i18n.MsgChangesSaved))
	})
}

// DELETE /api/v1/albums/:uid/like
//
// Parameters:
//   uid: string Album UID
func DislikeAlbum(router *gin.RouterGroup) {
	router.DELETE("/albums/:uid/like", func(c *gin.Context) {
		s := Auth(SessionID(c), acl.ResourceAlbums, acl.ActionLike)

		if s.Invalid() {
			AbortUnauthorized(c)
			return
		}

		conf := service.Config()
		id := c.Param("uid")
		album, err := query.AlbumByUID(id)

		if err != nil {
			Abort(c, http.StatusNotFound, i18n.ErrAlbumNotFound)
			return
		}

		album.AlbumFavorite = false
		conf.Db().Save(&album)

		UpdateClientConfig()
		PublishAlbumEvent(EntityUpdated, id, c)

		c.JSON(http.StatusOK, i18n.NewResponse(http.StatusOK, i18n.MsgChangesSaved))
	})
}

// POST /api/v1/albums/:uid/clone
func CloneAlbums(router *gin.RouterGroup) {
	router.POST("/albums/:uid/clone", func(c *gin.Context) {
		s := Auth(SessionID(c), acl.ResourceAlbums, acl.ActionUpdate)

		if s.Invalid() {
			AbortUnauthorized(c)
			return
		}

		a, err := query.AlbumByUID(c.Param("uid"))

		if err != nil {
			Abort(c, http.StatusNotFound, i18n.ErrAlbumNotFound)
			return
		}

		var f form.Selection

		if err := c.BindJSON(&f); err != nil {
			AbortBadRequest(c)
			return
		}

		var added []entity.PhotoAlbum

		for _, uid := range f.Albums {
			cloneAlbum, err := query.AlbumByUID(uid)

			if err != nil {
				log.Errorf("album: %s", err)
				continue
			}

			photos, err := query.AlbumPhotos(cloneAlbum, 10000)

			if err != nil {
				log.Errorf("album: %s", err)
				continue
			}

			added = append(added, a.AddPhotos(photos.UIDs())...)
		}

		if len(added) > 0 {
			event.SuccessMsg(i18n.MsgSelectionAddedTo, txt.Quote(a.Title()))

			PublishAlbumEvent(EntityUpdated, a.AlbumUID, c)
		}

		c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": i18n.Msg(i18n.MsgAlbumCloned), "album": a, "added": added})
	})
}

// POST /api/v1/albums/:uid/photos
func AddPhotosToAlbum(router *gin.RouterGroup) {
	router.POST("/albums/:uid/photos", func(c *gin.Context) {
		s := Auth(SessionID(c), acl.ResourceAlbums, acl.ActionUpdate)

		if s.Invalid() {
			AbortUnauthorized(c)
			return
		}

		var f form.Selection

		if err := c.BindJSON(&f); err != nil {
			AbortBadRequest(c)
			return
		}

		uid := c.Param("uid")
		a, err := query.AlbumByUID(uid)

		if err != nil {
			Abort(c, http.StatusNotFound, i18n.ErrAlbumNotFound)
			return
		}

		photos, err := query.PhotoSelection(f)

		if err != nil {
			log.Errorf("album: %s", err)
			AbortBadRequest(c)
			return
		}

		added := a.AddPhotos(photos.UIDs())

		if len(added) > 0 {
			if len(added) == 1 {
				event.SuccessMsg(i18n.MsgEntryAddedTo, txt.Quote(a.Title()))
			} else {
				event.SuccessMsg(i18n.MsgEntriesAddedTo, len(added), txt.Quote(a.Title()))
			}

			PublishAlbumEvent(EntityUpdated, a.AlbumUID, c)
		}

		c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": i18n.Msg(i18n.MsgChangesSaved), "album": a, "photos": photos.UIDs(), "added": added})
	})
}

// DELETE /api/v1/albums/:uid/photos
func RemovePhotosFromAlbum(router *gin.RouterGroup) {
	router.DELETE("/albums/:uid/photos", func(c *gin.Context) {
		s := Auth(SessionID(c), acl.ResourceAlbums, acl.ActionUpdate)

		if s.Invalid() {
			AbortUnauthorized(c)
			return
		}

		var f form.Selection

		if err := c.BindJSON(&f); err != nil {
			AbortBadRequest(c)
			return
		}

		if len(f.Photos) == 0 {
			Abort(c, http.StatusBadRequest, i18n.ErrNoItemsSelected)
			return
		}

		a, err := query.AlbumByUID(c.Param("uid"))

		if err != nil {
			Abort(c, http.StatusNotFound, i18n.ErrAlbumNotFound)
			return
		}

		removed := a.RemovePhotos(f.Photos)

		if len(removed) > 0 {
			if len(removed) == 1 {
				event.SuccessMsg(i18n.MsgEntryRemovedFrom, txt.Quote(a.Title()))
			} else {
				event.SuccessMsg(i18n.MsgEntriesRemovedFrom, len(removed), txt.Quote(txt.Quote(a.Title())))
			}

			PublishAlbumEvent(EntityUpdated, a.AlbumUID, c)
		}

		c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": i18n.Msg(i18n.MsgChangesSaved), "album": a, "photos": f.Photos, "removed": removed})
	})
}

// GET /api/v1/albums/:uid/dl
func DownloadAlbum(router *gin.RouterGroup) {
	router.GET("/albums/:uid/dl", func(c *gin.Context) {
		if InvalidDownloadToken(c) {
			AbortUnauthorized(c)
			return
		}

		start := time.Now()
		conf := service.Config()
		a, err := query.AlbumByUID(c.Param("uid"))

		if err != nil {
			Abort(c, http.StatusNotFound, i18n.ErrAlbumNotFound)
			return
		}

		p, err := query.AlbumPhotos(a, 10000)

		if err != nil {
			AbortEntityNotFound(c)
			return
		}

		zipPath := path.Join(conf.TempPath(), "album")
		zipToken := rnd.Token(3)
		zipBaseName := fmt.Sprintf("%s-%s.zip", strings.Title(a.AlbumSlug), zipToken)
		zipFileName := path.Join(zipPath, zipBaseName)

		if err := os.MkdirAll(zipPath, 0700); err != nil {
			log.Error(err)
			Abort(c, http.StatusInternalServerError, i18n.ErrCreateFolder)
			return
		}

		newZipFile, err := os.Create(zipFileName)

		if err != nil {
			log.Error(err)
			Abort(c, http.StatusInternalServerError, i18n.ErrCreateFile)
			return
		}

		defer newZipFile.Close()

		zipWriter := zip.NewWriter(newZipFile)
		defer func() { _ = zipWriter.Close() }()

		for _, f := range p {
			fileName := photoprism.FileName(f.FileRoot, f.FileName)

			fileAlias := f.ShareFileName()

			if fs.FileExists(fileName) {
				if err := addFileToZip(zipWriter, fileName, fileAlias); err != nil {
					log.Error(err)
					Abort(c, http.StatusInternalServerError, i18n.ErrCreateFile)
					return
				}
				log.Infof("album: added %s as %s", txt.Quote(f.FileName), txt.Quote(fileAlias))
			} else {
				log.Errorf("album: file %s is missing", txt.Quote(f.FileName))
			}
		}

		log.Infof("album: archive %s created in %s", txt.Quote(zipBaseName), time.Since(start))
		_ = zipWriter.Close()
		newZipFile.Close()

		if !fs.FileExists(zipFileName) {
			log.Errorf("could not find zip file: %s", zipFileName)
			c.Data(http.StatusNotFound, "image/svg+xml", photoIconSvg)
			return
		}

		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", zipBaseName))

		c.File(zipFileName)

		if err := os.Remove(zipFileName); err != nil {
			log.Errorf("album: could not remove %s (%s)", txt.Quote(zipFileName), err.Error())
		}
	})
}

// GET /api/v1/albums/:uid/t/:token/:type
//
// Parameters:
//   uid: string Album UID
//   type: string Thumbnail type, see photoprism.ThumbnailTypes
func AlbumThumbnail(router *gin.RouterGroup) {
	router.GET("/albums/:uid/t/:token/:type", func(c *gin.Context) {
		if InvalidPreviewToken(c) {
			c.Data(http.StatusForbidden, "image/svg+xml", albumIconSvg)
			return
		}

		start := time.Now()
		conf := service.Config()
		typeName := c.Param("type")
		uid := c.Param("uid")

		thumbType, ok := thumb.Types[typeName]

		if !ok {
			log.Errorf("album-thumbnail: invalid type %s", typeName)
			c.Data(http.StatusOK, "image/svg+xml", albumIconSvg)
			return
		}

		cache := service.Cache()
		cacheKey := fmt.Sprintf("album-thumbnail:%s:%s", uid, typeName)

		if cacheData, err := cache.Get(cacheKey); err == nil {
			log.Debugf("cache hit for %s [%s]", cacheKey, time.Since(start))

			var cached ThumbCache

			if err := json.Unmarshal(cacheData, &cached); err != nil {
				log.Errorf("album-thumbnail: %s not found", uid)
				c.Data(http.StatusOK, "image/svg+xml", albumIconSvg)
				return
			}

			if !fs.FileExists(cached.FileName) {
				log.Errorf("album-thumbnail: %s not found", uid)
				c.Data(http.StatusOK, "image/svg+xml", albumIconSvg)
				return
			}

			if c.Query("download") != "" {
				c.FileAttachment(cached.FileName, cached.ShareName)
			} else {
				c.File(cached.FileName)
			}

			return
		}

		f, err := query.AlbumCoverByUID(uid)

		if err != nil {
			log.Debugf("album-thumbnail: no photos yet, using generic image for %s", uid)
			c.Data(http.StatusOK, "image/svg+xml", albumIconSvg)
			return
		}

		fileName := photoprism.FileName(f.FileRoot, f.FileName)

		if !fs.FileExists(fileName) {
			log.Errorf("album-thumbnail: could not find original for %s", fileName)
			c.Data(http.StatusOK, "image/svg+xml", albumIconSvg)

			// Set missing flag so that the file doesn't show up in search results anymore.
			log.Warnf("album-thumbnail: %s is missing", txt.Quote(f.FileName))
			logError("album-thumbnail", f.Update("FileMissing", true))
			return
		}

		// Use original file if thumb size exceeds limit, see https://github.com/photoprism/photoprism/issues/157
		if thumbType.ExceedsLimit() && c.Query("download") == "" {
			log.Debugf("album-thumbnail: using original, size exceeds limit (width %d, height %d)", thumbType.Width, thumbType.Height)
			c.File(fileName)
			return
		}

		var thumbnail string

		if conf.ThumbUncached() || thumbType.OnDemand() {
			thumbnail, err = thumb.FromFile(fileName, f.FileHash, conf.ThumbPath(), thumbType.Width, thumbType.Height, thumbType.Options...)
		} else {
			thumbnail, err = thumb.FromCache(fileName, f.FileHash, conf.ThumbPath(), thumbType.Width, thumbType.Height, thumbType.Options...)
		}

		if err != nil {
			log.Errorf("album: %s", err)
			c.Data(http.StatusOK, "image/svg+xml", photoIconSvg)
			return
		} else if thumbnail == "" {
			log.Errorf("album-thumbnail: %s has empty thumb name - bug?", filepath.Base(fileName))
			c.Data(http.StatusOK, "image/svg+xml", albumIconSvg)
			return
		}

		if cached, err := json.Marshal(ThumbCache{thumbnail, f.ShareFileName()}); err == nil {
			logError("album-thumbnail", cache.Set(cacheKey, cached))
			log.Debugf("cached %s [%s]", cacheKey, time.Since(start))
		}

		if c.Query("download") != "" {
			c.FileAttachment(thumbnail, f.ShareFileName())
		} else {
			c.File(thumbnail)
		}
	})
}

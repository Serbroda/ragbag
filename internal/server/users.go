package server

import (
	"io"
	"net/http"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/fetch"
	"github.com/Serbroda/ragbag/internal/images"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/labstack/echo/v4"
)

func (s *Server) userDto(user *store.User) api.UserDto {
	admin := s.isAdmin(user)
	return api.UserDto{Id: user.ID, Username: user.Username, Email: user.Email, AvatarId: user.AvatarImageID,
		InstanceAdmin: &admin}
}

func (s *Server) Me(c echo.Context) error {
	user, err := s.store.UserByID(ctxOf(c), currentUser(c))
	if err != nil {
		return err
	}
	pin, err := s.store.PinState(ctxOf(c), user.ID)
	if err != nil {
		return err
	}
	dto := s.userDto(user)
	hasPin := pin.Hash != nil
	dto.HasPin = &hasPin
	return c.JSON(http.StatusOK, dto)
}

func (s *Server) SetAvatar(c echo.Context) error {
	contentType, data, err := s.readImage(c)
	if err != nil {
		return err
	}
	var user *store.User
	err = s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		previous, err := tx.UserByID(ctxOf(c), currentUser(c))
		if err != nil {
			return err
		}
		imageID, err := tx.CreateImage(ctxOf(c), contentType, data)
		if err != nil {
			return err
		}
		if err := tx.SetAvatar(ctxOf(c), previous.ID, &imageID); err != nil {
			return err
		}
		if err := tx.DeleteImage(ctxOf(c), previous.AvatarImageID); err != nil {
			return err
		}
		user, err = tx.UserByID(ctxOf(c), previous.ID)
		return err
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, s.userDto(user))
}

func (s *Server) RemoveAvatar(c echo.Context) error {
	var user *store.User
	err := s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		previous, err := tx.UserByID(ctxOf(c), currentUser(c))
		if err != nil {
			return err
		}
		if err := tx.SetAvatar(ctxOf(c), previous.ID, nil); err != nil {
			return err
		}
		if err := tx.DeleteImage(ctxOf(c), previous.AvatarImageID); err != nil {
			return err
		}
		user, err = tx.UserByID(ctxOf(c), previous.ID)
		return err
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, s.userDto(user))
}

// GetImage is public so <img> tags work without a token; ids are random and change with every upload.
func (s *Server) GetImage(c echo.Context, imageID api.ID) error {
	image, err := s.store.ImageByID(ctxOf(c), imageID)
	if err != nil {
		return orNotFound(err, "Image not found: %s", imageID)
	}
	header := c.Response().Header()
	header.Set("Cache-Control", "public, max-age=31536000, immutable")
	// Opened directly, the image must never be able to run anything on our origin
	header.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	header.Set("X-Content-Type-Options", "nosniff")
	return c.Blob(http.StatusOK, image.ContentType, image.Data)
}

// readImage takes the multipart field "file", or loads the image at "url" (SSRF-guarded), and normalizes it.
func (s *Server) readImage(c echo.Context) (string, []byte, error) {
	var data []byte
	if header, err := c.FormFile("file"); err == nil {
		if header.Size > images.MaxUploadBytes {
			return "", nil, badRequest("Images must be smaller than 5 MB")
		}
		file, err := header.Open()
		if err != nil {
			return "", nil, badRequest("The upload could not be read")
		}
		defer file.Close()
		if data, err = io.ReadAll(io.LimitReader(file, images.MaxUploadBytes+1)); err != nil {
			return "", nil, badRequest("The upload could not be read")
		}
		if len(data) > images.MaxUploadBytes {
			return "", nil, badRequest("Images must be smaller than 5 MB")
		}
	} else if rawURL := c.FormValue("url"); rawURL != "" {
		normalized, err := fetch.NormalizeURL(rawURL)
		if err != nil {
			return "", nil, badRequest("%s", err.Error())
		}
		res, err := s.fetch.Get(ctxOf(c), normalized, images.MaxUploadBytes)
		if err != nil {
			return "", nil, badRequest("The image could not be loaded from this URL (unreachable, not allowed or larger than 5 MB)")
		}
		data = res.Body
	} else {
		return "", nil, badRequest("Send an image file or the URL of one")
	}
	contentType, normalized, err := images.Normalize(data)
	if err != nil {
		return "", nil, badRequest("%s", err.Error())
	}
	return contentType, normalized, nil
}

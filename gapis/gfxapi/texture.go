// Copyright (C) 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gfxapi

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/data/protoutil"
	"github.com/google/gapid/core/image"
)

func (l *CubemapLevel) faces() [6]*image.Info2D {
	return [6]*image.Info2D{
		l.NegativeX,
		l.PositiveX,
		l.NegativeY,
		l.PositiveY,
		l.NegativeZ,
		l.PositiveZ,
	}
}

func (l *CubemapLevel) setFaces(faces [6]*image.Info2D) {
	l.NegativeX,
		l.PositiveX,
		l.NegativeY,
		l.PositiveY,
		l.NegativeZ,
		l.PositiveZ = faces[0], faces[1], faces[2], faces[3], faces[4], faces[5]
}

type imageMatcher struct {
	best          *image.Info2D
	score         uint32
	width, height uint32
}

func (m *imageMatcher) consider(i *image.Info2D) {
	if i == nil {
		return
	}

	if m.best == nil {
		m.score = 0xffffffff
	}
	dw, dh := i.Width-m.width, i.Height-m.height
	score := dw*dw + dh*dh
	if m.score > score {
		m.score = score
		m.best = i
	}
}

// Thumbnail returns the image that most closely matches the desired size.
func (t *Texture2D) Thumbnail(ctx context.Context, w, h uint32) (*image.Info2D, error) {
	m := imageMatcher{width: w, height: h}
	for _, l := range t.Levels {
		m.consider(l)
	}

	return m.best, nil
}

// Interface compliance check
var _ = image.Convertable((*Texture2D)(nil))

// ConvertTo returns this Texture2D with each mip-level converted to the requested format.
func (t *Texture2D) ConvertTo(ctx context.Context, f *image.Format) (interface{}, error) {
	out := &Texture2D{
		Levels: make([]*image.Info2D, len(t.Levels)),
	}
	for i, m := range t.Levels {
		if obj, err := m.Convert(ctx, f); err == nil {
			out.Levels[i] = obj
		} else {
			return nil, err
		}
	}
	return out, nil
}

// Thumbnail returns the image that most closely matches the desired size.
func (t *Cubemap) Thumbnail(ctx context.Context, w, h uint32) (*image.Info2D, error) {
	m := imageMatcher{width: w, height: h}

	for _, l := range t.Levels {
		m.consider(l.NegativeX)
		m.consider(l.PositiveX)
		m.consider(l.NegativeY)
		m.consider(l.PositiveY)
		m.consider(l.NegativeZ)
		m.consider(l.PositiveZ)
	}

	return m.best, nil
}

// Interface compliance check
var _ = image.Convertable((*Cubemap)(nil))

// ConvertTo returns this Cubemap with each mip-level face converted to the requested format.
func (t *Cubemap) ConvertTo(ctx context.Context, f *image.Format) (interface{}, error) {
	out := &Cubemap{
		Levels: make([]*CubemapLevel, len(t.Levels)),
	}
	for i, m := range t.Levels {
		out.Levels[i] = &CubemapLevel{}
		dst, src := out.Levels[i].faces(), m.faces()
		for j, srcFace := range src {
			if srcFace == nil {
				continue
			}

			cnvFace, err := srcFace.Convert(ctx, f)
			if err != nil {
				return nil, err
			}
			dst[j] = cnvFace
		}
		out.Levels[i].setFaces(dst)
	}
	return out, nil
}

// Interface compliance check
var _ = image.Convertable((*Texture)(nil))
var _ = image.Thumbnailer((*Texture)(nil))

// ConvertTo returns this Texture2D with each mip-level converted to the requested format.
func (t *Texture) ConvertTo(ctx context.Context, f *image.Format) (interface{}, error) {
	data := protoutil.OneOf(t.Type)
	if c, ok := data.(image.Convertable); ok {
		data, err := c.ConvertTo(ctx, f)
		if err != nil {
			return nil, err
		}
		return NewTexture(data), nil
	}
	return nil, nil
}

// Thumbnail returns the image that most closely matches the desired size.
func (t *Texture) Thumbnail(ctx context.Context, w, h uint32) (*image.Info2D, error) {
	data := protoutil.OneOf(t.Type)
	if t, ok := data.(image.Thumbnailer); ok {
		return t.Thumbnail(ctx, w, h)
	}
	return nil, nil
}

// NewTexture returns a new *ResourceData with the specified texture.
func NewTexture(t interface{}) *Texture {
	switch t := t.(type) {
	case *Texture2D:
		return &Texture{Type: &Texture_Texture_2D{t}}
	case *Cubemap:
		return &Texture{Type: &Texture_Cubemap{t}}
	default:
		panic(fmt.Errorf("%T is not a Texture type", t))
	}
}

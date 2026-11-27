package singleinstance

import (
	"context"
	"fmt"
	"time"

	"github.com/godbus/dbus/v5"
)

// Nama D-Bus yang dipakai instance pertama untuk menerima permintaan fokus.
const (
	BusName    = "org.cleanlie.Cleanlie"
	ObjectPath = "/org/cleanlie/Cleanlie"
	Interface  = "org.cleanlie.Cleanlie"
)

// Activate meminta instance yang sedang berjalan untuk menampilkan window-nya.
// Dipakai instance kedua sebelum keluar; error berarti instance lama tidak bisa
// dihubungi (mis. session bus tidak tersedia).
func Activate() error {
	conn, err := dbus.SessionBus()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	obj := conn.Object(BusName, dbus.ObjectPath(ObjectPath))
	return obj.CallWithContext(ctx, Interface+".Focus", 0).Err
}

// FocusServer adalah listener D-Bus milik instance pertama.
type FocusServer struct {
	conn *dbus.Conn
}

// ServeFocus mengekspor method Focus di session bus. onFocus dipanggil saat
// instance kedua meminta fokus. Error dikembalikan bila bus/nama tidak tersedia;
// pemanggil boleh mengabaikannya dan tetap menjalankan GUI.
func ServeFocus(onFocus func()) (*FocusServer, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, err
	}
	if err := conn.ExportMethodTable(map[string]interface{}{
		"Focus": func() *dbus.Error {
			if onFocus != nil {
				onFocus()
			}
			return nil
		},
	}, dbus.ObjectPath(ObjectPath), Interface); err != nil {
		return nil, err
	}
	reply, err := conn.RequestName(BusName, dbus.NameFlagDoNotQueue)
	if err != nil {
		return nil, err
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return nil, fmt.Errorf("nama D-Bus %s sudah dipakai", BusName)
	}
	return &FocusServer{conn: conn}, nil
}

// Close melepas nama D-Bus. Koneksi session bus dipakai bersama, jadi tidak
// ditutup di sini.
func (s *FocusServer) Close() error {
	if s == nil || s.conn == nil {
		return nil
	}
	_, err := s.conn.ReleaseName(BusName)
	s.conn = nil
	return err
}

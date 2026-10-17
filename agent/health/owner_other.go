//go:build !unix

package health

import "os"

func connectionMarkerTrusted(info os.FileInfo) bool { return info.Mode().IsRegular() }

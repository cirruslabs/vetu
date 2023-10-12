package oci

import "github.com/dustin/go-humanize"

const (
	MediaTypeVetuConfig    = "application/vnd.cirruslabs.vetu.config.v1"
	MediaTypeVetuKernel    = "application/vnd.cirruslabs.vetu.kernel.v1"
	MediaTypeVetuInitramfs = "application/vnd.cirruslabs.vetu.initramfs.v1"
	MediaTypeVetuDisk      = "application/vnd.cirruslabs.vetu.disk.v1"

	AnnotationName               = "org.cirruslabs.vetu.name"
	AnnotationUncompressedSize   = "org.cirruslabs.vetu.uncompressed-size"
	AnnotationUncompressedDigest = "org.cirruslabs.vetu.uncompressed-digest"

	targetDiskLayerSizeBytes = 500 * humanize.MByte
)

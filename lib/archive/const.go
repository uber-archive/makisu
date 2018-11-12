package archive

const _whiteoutPrefix = ".wh."

// WhiteoutMetaPrefix means it is AUFS metadata, not for removing files.
// Should be ignored during untar.
// TODO: There could be hardlinks pointing to files under /.wh..wh.plnk.
const _whiteoutMetaPrefix = _whiteoutPrefix + _whiteoutPrefix

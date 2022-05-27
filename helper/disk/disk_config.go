package disk

const (
	FormatThin        = "thin"
	FormatFlat        = "flat"
	FormatThink       = "think"
	FormatNativeThink = "nativeThink"
)

const (
	ModePersistent               = "persistent"                // Changes are appended to the redo log; you revoke changes by removing the undo log.
	ModeIndependentPersistent    = "Independent_persistent"    // Same as nonpersistent, but not affected by snapshots.
	ModeIndependentNonpersistent = "Independent_nonpersistent" // Same as persistent, but not affected by snapshots.
	ModeNonpersistent            = "Nonpersistent"             // Changes to virtual disk are made to a redo log and discarded at power off.
	ModeUndoable                 = "Undoable"                  // Changes are immediately and permanently written to the virtual disk.
	ModeAppend                   = "Append"                    // Changes are made to a redo log, but you are given the option to commit or undo.
)

const (
	SharingMultiWriter = "sharingNone" //  The virtual disk is shared between multiple virtual machines.
	SharingNone        = "sharingNone" //  The virtual disk is not shared.
)

var Formats []Format
var FormatMapping = map[string]*Format{}

type Format struct {
	Label           string
	EagerlyScrub    *bool
	ThinProvisioned *bool
}

func init() {
	t := true
	f := false

	thinFormat := Format{Label: FormatThin, EagerlyScrub: nil, ThinProvisioned: &t}
	Formats = append(Formats, thinFormat)
	FormatMapping[FormatThin] = &thinFormat

	flatFormat := Format{Label: FormatFlat, EagerlyScrub: nil, ThinProvisioned: &f}
	Formats = append(Formats, flatFormat)
	FormatMapping[FormatFlat] = &flatFormat

	thinkFormat := Format{Label: FormatThink, EagerlyScrub: &t, ThinProvisioned: &f}
	Formats = append(Formats, thinkFormat)
	FormatMapping[FormatThink] = &thinkFormat

	nativeThinkFormat := Format{Label: FormatNativeThink, EagerlyScrub: nil, ThinProvisioned: nil}
	Formats = append(Formats, nativeThinkFormat)
	FormatMapping[FormatNativeThink] = &nativeThinkFormat
}

func GetFormat(eagerlyScrub, thinProvisioned *bool) string {
	if eagerlyScrub == nil && thinProvisioned != nil && *thinProvisioned {
		return FormatThin
	} else if eagerlyScrub == nil && thinProvisioned != nil && !*thinProvisioned {
		return FormatFlat
	} else if eagerlyScrub != nil && *eagerlyScrub && thinProvisioned != nil && !*thinProvisioned {
		return FormatThink
	} else if eagerlyScrub == nil && thinProvisioned == nil {
		return FormatNativeThink
	} else {
		return ""
	}
}

func GetFormats(datastoreType string) []string {
	switch datastoreType {
	case "NFS", "NFS41":
		return []string{FormatThin}
	case "VVOL":
		return []string{FormatThin, FormatNativeThink}
	case "VSAN":
		return nil
	case "VMFS":
		return []string{FormatThin, FormatFlat, FormatNativeThink}

	}
	return nil
}

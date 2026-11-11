//go:build no_self_update || ios

package ghupdate

// SelfUpdateDisabled reports whether self-update was disabled at build time
// with the no_self_update build tag. Package managers that handle upgrades
// themselves (e.g. Homebrew) should build with this tag.
// iOS always uses the port's signed package/installer lifecycle: upstream
// archives cannot preserve its A7 runtime, signatures or rollback checkpoint.
const SelfUpdateDisabled = true

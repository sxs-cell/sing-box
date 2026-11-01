//go:build ferry_mobile

package libbox

func OpenNativeShellSession(
	shell, cwd string,
	args, environ StringIterator,
	term string,
	rows, cols, uid, gid int32,
	groups Int32Iterator,
) (ShellSession, error) {
	return nil, errFerryUnsupported
}

func OpenNativePipeSession(
	shell, cwd string,
	args, environ StringIterator,
	uid, gid int32,
	groups Int32Iterator,
) (ShellSession, error) {
	return nil, errFerryUnsupported
}

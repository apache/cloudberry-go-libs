package dbconn

import (
	"regexp"
	"strings"

	"github.com/blang/semver"
	"github.com/cloudberrydb/gp-common-go-libs/gplog"
)

// DBType represents the type of database
type DBType int

const (
	Unknown DBType = iota
	GPDB           // Greenplum Database
	CBDB           // Apache Cloudberry Database
)

const (
	gpdbPattern = `\(Greenplum Database ([0-9]+\.[0-9]+\.[0-9]+)[^)]*\)`
	cbdbPattern = `\(Apache Cloudberry ([0-9]+\.[0-9]+\.[0-9]+)[^)]*\)`
)

// String provides string representation of DBType
func (t DBType) String() string {
	switch t {
	case GPDB:
		return "Greenplum Database"
	case CBDB:
		return "Apache Cloudberry"
	default:
		return "Unknown Database"
	}
}

// GPDBVersion represents version information for a database
type GPDBVersion struct {
	VersionString string
	SemVer        semver.Version
	Type          DBType
}

/*
 * This constructor is intended as a convenience function for testing and
 * setting defaults; the dbconn.Connect function will automatically initialize
 * the version of the database to which it is connecting.
 *
 * The versionStr argument here should be a semantic version in the form X.Y.Z,
 * not a GPDB version string like the one returned by "SELECT version()".  If
 * an invalid semantic version is passed, that is considered programmer error
 * and the function will panic.
 */
func NewVersion(versionStr string) GPDBVersion {
	version := GPDBVersion{
		VersionString: versionStr,
		SemVer:        semver.MustParse(versionStr),
		Type:          GPDB, // Default to GPDB for tests
	}
	return version
}

// InitializeVersion parses database version string and returns version information
// It can distinguish between Greenplum Database and Apache Cloudberry Database.
func InitializeVersion(dbconn *DBConn) (dbversion GPDBVersion, err error) {
	err = dbconn.Get(&dbversion, "SELECT pg_catalog.version() AS versionstring")
	if err != nil {
		return
	}

	// Determine database type and parse version
	dbversion.ParseVersionInfo(dbversion.VersionString)

	gplog.Info("Initialized database version - Full Version: %s, Database Type: %s, Semantic Version: %s",
		dbversion.VersionString, dbversion.Type, dbversion.SemVer)
	return
}

func (dbversion *GPDBVersion) ParseVersionInfo(versionString string) {
	dbversion.VersionString = versionString
	dbversion.Type = Unknown

	// Try to match each database type.
	// We check for Apache Cloudberry first as its string may be a superset of others in the future.
	if ver, ok := dbversion.extractVersion(cbdbPattern); ok {
		dbversion.Type = CBDB
		dbversion.SemVer = ver
	} else if ver, ok := dbversion.extractVersion(gpdbPattern); ok {
		dbversion.Type = GPDB
		dbversion.SemVer = ver
	}
}

func (dbversion GPDBVersion) extractVersion(pattern string) (semver.Version, bool) {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(dbversion.VersionString)
	if len(matches) < 2 {
		return semver.Version{}, false
	}

	ver, err := semver.Make(matches[1])
	if err != nil {
		return semver.Version{}, false
	}
	return ver, true
}

func (dbversion GPDBVersion) StringToSemVerRange(versionStr string) semver.Range {
	numDigits := len(strings.Split(versionStr, "."))
	if numDigits < 3 {
		versionStr += ".x"
	}
	validRange := semver.MustParseRange(versionStr)
	return validRange
}

func (dbversion GPDBVersion) Before(targetVersion string) bool {
	validRange := dbversion.StringToSemVerRange("<" + targetVersion)
	return validRange(dbversion.SemVer)
}

func (dbversion GPDBVersion) AtLeast(targetVersion string) bool {
	validRange := dbversion.StringToSemVerRange(">=" + targetVersion)
	return validRange(dbversion.SemVer)
}

func (dbversion GPDBVersion) Is(targetVersion string) bool {
	validRange := dbversion.StringToSemVerRange("==" + targetVersion)
	return validRange(dbversion.SemVer)
}

func (dbversion GPDBVersion) IsGPDB() bool {
	return dbversion.Type == GPDB
}

func (dbversion GPDBVersion) IsCBDB() bool {
	return dbversion.Type == CBDB
}

func (srcVersion GPDBVersion) Equals(destVersion GPDBVersion) bool {
	if srcVersion.Type != destVersion.Type {
		return false
	}

	return srcVersion.SemVer.Major == destVersion.SemVer.Major
}

package dbconn_test

import (
	"github.com/blang/semver"
	"github.com/cloudberrydb/gp-common-go-libs/dbconn"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("dbconn/version tests", func() {
	// Fake versions for testing
	fakeGPDB43 := dbconn.GPDBVersion{VersionString: "4.3.0.0", SemVer: semver.MustParse("4.3.0"), Type: dbconn.GPDB}
	fakeGPDB5 := dbconn.GPDBVersion{VersionString: "5.0.0", SemVer: semver.MustParse("5.0.0"), Type: dbconn.GPDB}
	fakeGPDB51 := dbconn.GPDBVersion{VersionString: "5.1.0", SemVer: semver.MustParse("5.1.0"), Type: dbconn.GPDB}
	fakeCBDB2 := dbconn.GPDBVersion{VersionString: "2.0.0", SemVer: semver.MustParse("2.0.0"), Type: dbconn.CBDB}

	Describe("ParseVersionInfo", func() {
		It("parses a GPDB version string", func() {
			versionStr := "PostgreSQL 12.12 (Greenplum Database 7.0.0 build commit:bf073b87c0bac9759631746dca1c4c895a304afb) on x86_64-pc-linux-gnu"
			dbVersion := dbconn.GPDBVersion{}
			dbVersion.ParseVersionInfo(versionStr)
			Expect(dbVersion.Type).To(Equal(dbconn.GPDB))
			Expect(dbVersion.SemVer.String()).To(Equal("7.0.0"))
			Expect(dbVersion.IsGPDB()).To(BeTrue())
			Expect(dbVersion.IsCBDB()).To(BeFalse())
		})
		It("parses an Apache Cloudberry version string", func() {
			versionStr := "PostgreSQL 14.4 (Apache Cloudberry 2.0.0 build commit:a071e3f8aa638786f01bbd08307b6474a1ba7890) on x86_64-pc-linux-gnu"
			dbVersion := dbconn.GPDBVersion{}
			dbVersion.ParseVersionInfo(versionStr)
			Expect(dbVersion.Type).To(Equal(dbconn.CBDB))
			Expect(dbVersion.SemVer.String()).To(Equal("2.0.0"))
			Expect(dbVersion.IsCBDB()).To(BeTrue())
			Expect(dbVersion.IsGPDB()).To(BeFalse())
		})
		It("handles an unknown version string", func() {
			versionStr := "Some Other Database 1.0.0"
			dbVersion := dbconn.GPDBVersion{}
			dbVersion.ParseVersionInfo(versionStr)
			Expect(dbVersion.Type).To(Equal(dbconn.Unknown))
			Expect(dbVersion.SemVer.String()).To(Equal("0.0.0"))
		})
	})
	Describe("StringToSemVerRange", func() {
		v400 := semver.MustParse("4.0.0")
		v500 := semver.MustParse("5.0.0")
		v510 := semver.MustParse("5.1.0")
		v501 := semver.MustParse("5.0.1")
		It(`turns "=5" into a range matching 5.x`, func() {
			resultRange := fakeGPDB5.StringToSemVerRange("=5")
			Expect(resultRange(v400)).To(BeFalse())
			Expect(resultRange(v500)).To(BeTrue())
			Expect(resultRange(v510)).To(BeTrue())
			Expect(resultRange(v501)).To(BeTrue())
		})
		It(`turns "=5.0" into a range matching 5.0.x`, func() {
			resultRange := fakeGPDB5.StringToSemVerRange("=5.0")
			Expect(resultRange(v400)).To(BeFalse())
			Expect(resultRange(v500)).To(BeTrue())
			Expect(resultRange(v510)).To(BeFalse())
			Expect(resultRange(v501)).To(BeTrue())
		})
		It(`turns "=5.0.0" into a range matching 5.0.0`, func() {
			resultRange := fakeGPDB5.StringToSemVerRange("=5.0.0")
			Expect(resultRange(v400)).To(BeFalse())
			Expect(resultRange(v500)).To(BeTrue())
			Expect(resultRange(v510)).To(BeFalse())
			Expect(resultRange(v501)).To(BeFalse())
		})
	})
	Describe("Before", func() {
		It("returns true when comparing 4.3 to 5", func() {
			connection.Version = fakeGPDB43
			result := connection.Version.Before("5")
			Expect(result).To(BeTrue())
		})
		It("returns true when comparing 5 to 5.1", func() {
			connection.Version = fakeGPDB5
			result := connection.Version.Before("5.1")
			Expect(result).To(BeTrue())
		})
		It("returns false when comparing 5 to 5", func() {
			connection.Version = fakeGPDB5
			result := connection.Version.Before("5")
			Expect(result).To(BeFalse())
		})
	})
	Describe("AtLeast", func() {
		It("returns true when comparing 5 to 4.3", func() {
			connection.Version = fakeGPDB5
			result := connection.Version.AtLeast("4")
			Expect(result).To(BeTrue())
		})
		It("returns true when comparing 5 to 5", func() {
			connection.Version = fakeGPDB5
			result := connection.Version.AtLeast("5")
			Expect(result).To(BeTrue())
		})
		It("returns true when comparing 5.1 to 5.0", func() {
			connection.Version = fakeGPDB51
			result := connection.Version.AtLeast("5")
			Expect(result).To(BeTrue())
		})
		It("returns false when comparing 4.3 to 5", func() {
			connection.Version = fakeGPDB43
			result := connection.Version.AtLeast("5")
			Expect(result).To(BeFalse())
		})
		It("returns false when comparing 5.0 to 5.1", func() {
			connection.Version = fakeGPDB5
			result := connection.Version.AtLeast("5.1")
			Expect(result).To(BeFalse())
		})
	})
	Describe("Is", func() {
		It("returns true when comparing 5 to 5", func() {
			connection.Version = fakeGPDB5
			result := connection.Version.Is("5")
			Expect(result).To(BeTrue())
		})
		It("returns true when comparing 5.1 to 5", func() {
			connection.Version = fakeGPDB51
			result := connection.Version.Is("5")
			Expect(result).To(BeTrue())
		})
		It("returns false when comparing 5.0 to 5.1", func() {
			connection.Version = fakeGPDB5
			result := connection.Version.Is("5.1")
			Expect(result).To(BeFalse())
		})
		It("returns false when comparing 4.3 to 5", func() {
			connection.Version = fakeGPDB43
			result := connection.Version.Is("5")
			Expect(result).To(BeFalse())
		})
	})
	Describe("Equals", func() {
		It("returns false if db types are different", func() {
			Expect(fakeGPDB5.Equals(fakeCBDB2)).To(BeFalse())
		})
		It("returns true if db types are same and major version is same", func() {
			anotherGPDB5 := dbconn.GPDBVersion{SemVer: semver.MustParse("5.2.0"), Type: dbconn.GPDB}
			Expect(fakeGPDB5.Equals(anotherGPDB5)).To(BeTrue())
		})
		It("returns false if db types are same but major version is different", func() {
			Expect(fakeGPDB5.Equals(fakeGPDB43)).To(BeFalse())
		})
	})
})

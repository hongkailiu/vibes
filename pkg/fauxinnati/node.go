package fauxinnati

import (
	"fmt"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/sets"
)

type NodeBuilder struct {
	queriedVersion semver.Version
	version        semver.Version
	channels       []string
	architecture   string

	client         Client
	getLatest      func(client Client, major, minor uint64) (semver.Version, error)
	digestResolver DigestResolver
}

type DigestResolver interface {
	getDigest(client Client, tag string) (string, error)
	getRepository() string
}

// resolvePatchVersion replaces a synthesized version with a real released one where it can, so
// that generated graphs point at payloads that actually exist. It looks up the head of the
// candidate channel for version's major.minor and returns it only when all of the following
// hold:
//
//   - the lookup succeeded;
//   - the head is in the same major.minor as version;
//   - the head is newer than queriedVersion, so the graph never offers an update to a version
//     the cluster is already at or past;
//   - the head is not newer than version, so it stays within the topology the caller laid out.
//
// In every other case version is returned unchanged, and the first condition to fail decides
// which warning is logged.
func resolvePatchVersion(client Client, queriedVersion, version semver.Version, getLatest func(client Client, major, minor uint64) (semver.Version, error)) semver.Version {
	latest, err := getLatest(client, version.Major, version.Minor)
	if err != nil {
		logrus.WithError(err).WithField("version", version).Warning("Fail to find the latest version")
		return version
	}

	log := logrus.WithFields(logrus.Fields{"version": version, "queriedVersion": queriedVersion, "latest": latest})
	switch {
	case latest.Major != version.Major || latest.Minor != version.Minor:
		log.Warning("The latest version in the candidate channel is not in the required minor")
	case latest.LTE(queriedVersion):
		log.Warning("The latest version is not greater than the queried version")
	case latest.GT(version):
		log.Debug("The latest version is newer than the version to synthesize")
	default:
		log.Debug("Use the latest patch version")
		return latest
	}
	return version
}

func (b *NodeBuilder) Build() Node {
	version := resolvePatchVersion(b.client, b.queriedVersion, b.version, b.getLatest)
	suffix := b.architecture
	switch b.architecture {
	case "":
		logrus.Debug("No architecture specified. Using default to resolve the image digest")
		suffix = "x86_64"
	case "amd64":
		suffix = "x86_64"
	case "arm64":
		suffix = "aarch64"
	}
	tag := fmt.Sprintf("%s-%s", version.String(), suffix)
	digest, err := b.digestResolver.getDigest(b.client, tag)
	if err != nil {
		digest = fmt.Sprintf("sha256:%064x", version.Major*1000000+version.Minor*1000+version.Patch)
	}
	metadata := map[string]string{
		"io.openshift.upgrades.graph.release.channels":    strings.Join(sets.List[string](sets.New[string](b.channels...)), ","),
		"io.openshift.upgrades.graph.release.manifestref": digest,
		"url": fmt.Sprintf("https://access.redhat.com/errata/RHSA-2024:%05d", version.Major*1000+version.Minor*100+version.Patch),
	}
	if b.architecture == "multi" {
		metadata["release.openshift.io/architecture"] = b.architecture
	}
	return Node{
		Version:  version,
		Image:    fmt.Sprintf("%s@%s", b.digestResolver.getRepository(), digest),
		Metadata: metadata,
	}
}

func (b *NodeBuilder) WithChannels(channels []string) *NodeBuilder {
	b.channels = channels
	return b
}

func (b *NodeBuilder) WithArchitecture(architecture string) *NodeBuilder {
	b.architecture = architecture
	return b
}

func (b *NodeBuilder) WithVersion(version semver.Version) *NodeBuilder {
	b.version = version
	return b
}

func (b *NodeBuilder) WithClient(client Client) *NodeBuilder {
	b.client = client
	return b
}

func (b *NodeBuilder) withQueriedVersion(queriedVersion semver.Version) *NodeBuilder {
	b.queriedVersion = queriedVersion
	return b
}

func (b *NodeBuilder) withGetLatest(getLatest func(client Client, major, minor uint64) (semver.Version, error)) *NodeBuilder {
	b.getLatest = getLatest
	return b
}

func (b *NodeBuilder) WithDigestResolver(digestResolver DigestResolver) *NodeBuilder {
	b.digestResolver = digestResolver
	return b
}

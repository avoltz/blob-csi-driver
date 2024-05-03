//go:build tools

package fakedeps

/* These dependencies are here to turn indirect depenedencies that we are not able to control the version of
 * into direct dependencies that we can. This is sometimes needed to resolve component governance issues. */

import (
	_ "go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful"
)

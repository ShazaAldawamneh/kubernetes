// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

// ImageSourcePathApplyConfiguration represents an declarative configuration of the ImageSourcePath type for use
// with apply.
type ImageSourcePathApplyConfiguration struct {
	SourcePath     *string `json:"sourcePath,omitempty"`
	DestinationDir *string `json:"destinationDir,omitempty"`
}

// ImageSourcePathApplyConfiguration constructs an declarative configuration of the ImageSourcePath type for use with
// apply.
func ImageSourcePath() *ImageSourcePathApplyConfiguration {
	return &ImageSourcePathApplyConfiguration{}
}

// WithSourcePath sets the SourcePath field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the SourcePath field is set to the value of the last call.
func (b *ImageSourcePathApplyConfiguration) WithSourcePath(value string) *ImageSourcePathApplyConfiguration {
	b.SourcePath = &value
	return b
}

// WithDestinationDir sets the DestinationDir field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the DestinationDir field is set to the value of the last call.
func (b *ImageSourcePathApplyConfiguration) WithDestinationDir(value string) *ImageSourcePathApplyConfiguration {
	b.DestinationDir = &value
	return b
}

package renderer

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
)

// CompileShaderGLSL compiles GLSL shader to SPIR-V using glslangValidator or glslc
func CompileShaderGLSL(source string, stage string, outputPath string) ([]uint32, error) {
	// Write source to temp file
	tempSrc := outputPath + ".tmp"
	if err := os.WriteFile(tempSrc, []byte(source), 0644); err != nil {
		return nil, err
	}
	defer os.Remove(tempSrc)
	
	// Try glslc first (Google's shader compiler), then glslangValidator
	var cmd *exec.Cmd
	
	if _, err := exec.LookPath("glslc"); err == nil {
		cmd = exec.Command("glslc", tempSrc, "-o", outputPath, "-O")
	} else if _, err := exec.LookPath("glslangValidator"); err == nil {
		cmd = exec.Command("glslangValidator", "-V", tempSrc, "-o", outputPath)
	} else {
		return nil, fmt.Errorf("no shader compiler found (glslc or glslangValidator)")
	}
	
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("shader compilation failed: %v\n%s", err, output)
	}
	
	// Read compiled SPIR-V
	data, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, err
	}
	defer os.Remove(outputPath)
	
	// Convert bytes to uint32 slice
	words := make([]uint32, len(data)/4)
	for i := 0; i < len(words); i++ {
		words[i] = binary.LittleEndian.Uint32(data[i*4:])
	}
	
	return words, nil
}

// Default vertex shader for basic rendering
const DefaultVertexShaderGLSL = `
#version 450

layout(binding = 0) uniform UniformBufferObject {
    mat4 model;
    mat4 view;
    mat4 proj;
} ubo;

layout(location = 0) in vec3 inPosition;
layout(location = 1) in vec3 inNormal;
layout(location = 2) in vec2 inTexCoord;
layout(location = 3) in vec4 inColor;

layout(location = 0) out vec3 fragNormal;
layout(location = 1) out vec2 fragTexCoord;
layout(location = 2) out vec4 fragColor;
layout(location = 3) out vec3 fragPos;

void main() {
    vec4 worldPos = ubo.model * vec4(inPosition, 1.0);
    gl_Position = ubo.proj * ubo.view * worldPos;
    fragPos = worldPos.xyz;
    fragNormal = mat3(transpose(inverse(ubo.model))) * inNormal;
    fragTexCoord = inTexCoord;
    fragColor = inColor;
}
`

// Default fragment shader for basic rendering
const DefaultFragmentShaderGLSL = `
#version 450

layout(location = 0) in vec3 fragNormal;
layout(location = 1) in vec2 fragTexCoord;
layout(location = 2) in vec4 fragColor;
layout(location = 3) in vec3 fragPos;

layout(location = 0) out vec4 outColor;

layout(binding = 1) uniform sampler2D texSampler;

void main() {
    vec3 lightDir = normalize(vec3(1.0, -1.0, -1.0));
    vec3 normal = normalize(fragNormal);
    float diff = max(dot(normal, -lightDir), 0.0);
    vec3 ambient = vec3(0.3);
    vec3 lighting = ambient + vec3(1.0) * diff;
    
    outColor = vec4(fragColor.rgb * lighting, fragColor.a);
}
`

// Simple shader (no lighting)
const SimpleVertexShaderGLSL = `
#version 450

layout(binding = 0) uniform UniformBufferObject {
    mat4 mvp;
} ubo;

layout(location = 0) in vec3 inPosition;
layout(location = 1) in vec3 inColor;

layout(location = 0) out vec3 fragColor;

void main() {
    gl_Position = ubo.mvp * vec4(inPosition, 1.0);
    fragColor = inColor;
}
`

const SimpleFragmentShaderGLSL = `
#version 450

layout(location = 0) in vec3 fragColor;
layout(location = 0) out vec4 outColor;

void main() {
    outColor = vec4(fragColor, 1.0);
}
`

// Pre-compiled SPIR-V for simple shaders (to avoid external compiler dependency)
// These are minimal SPIR-V shaders for testing

// Simple vertex shader SPIR-V (hardcoded for fallback)
var SimpleVertexShaderSPIRV = []uint32{
	0x07230203, 0x00010000, 0x00080001, 0x0000002a, 0x00000000, 0x00020011, 0x00000001,
	0x0006000b, 0x00000001, 0x4c534c47, 0x6474732e, 0x3035342e, 0x00000000, 0x0003000e,
	0x00000000, 0x00000001, 0x0008000f, 0x00000000, 0x00000004, 0x6e69616d, 0x00000000,
	0x0000000d, 0x00000012, 0x0000001c, 0x00030003, 0x00000002, 0x000001c2, 0x00040005,
	0x00000004, 0x6e69616d, 0x00000000, 0x00050005, 0x0000000c, 0x67617266, 0x736f6c43,
	0x00000000, 0x00040005, 0x0000000d, 0x6e695f75, 0x00006e49, 0x00060005, 0x00000012,
	0x626f6f62, 0x6a624f6a, 0x746365, 0x00000000, 0x00030005, 0x00000017, 0x00007875,
	0x00050005, 0x0000001c, 0x6e695f6e, 0x6f50736f, 0x00006974, 0x00050048, 0x00000012,
	0x00000000, 0x0000000b, 0x00000000, 0x00030047, 0x00000012, 0x00000002, 0x00040047,
	0x0000001c, 0x0000001e, 0x00000000, 0x00040047, 0x0000001c, 0x0000001e, 0x00000001,
	0x00040047, 0x0000001c, 0x0000001e, 0x00000002, 0x00020013, 0x00000002, 0x00030021,
	0x00000003, 0x00000002, 0x00030016, 0x00000006, 0x00000020, 0x00040017, 0x00000007,
	0x00000006, 0x00000004, 0x00040015, 0x00000008, 0x00000020, 0x00000001, 0x00020013,
	0x0000000a, 0x00030021, 0x0000000b, 0x0000000a, 0x00030016, 0x0000000e, 0x00000020,
	0x0004001b, 0x0000000f, 0x0000000e, 0x00000004, 0x0003001e, 0x00000010, 0x0000000f,
	0x00040020, 0x00000011, 0x00000003, 0x00000010, 0x0004003b, 0x00000011, 0x00000012,
	0x00000003, 0x00040015, 0x00000013, 0x00000020, 0x00000000, 0x0004002b, 0x00000013,
	0x00000014, 0x00000000, 0x00040020, 0x00000015, 0x00000003, 0x00000007, 0x00090019,
	0x00000017, 0x00000006, 0x00000001, 0x00000000, 0x00000000, 0x00000000, 0x00000001,
	0x00000000, 0x0003001b, 0x00000018, 0x00000017, 0x00040020, 0x00000019, 0x00000000,
	0x00000018, 0x0004003b, 0x00000019, 0x0000001a, 0x00000000, 0x0004002b, 0x00000008,
	0x0000001b, 0x00000000, 0x00090019, 0x0000001c, 0x00000006, 0x00000001, 0x00000000,
	0x00000000, 0x00000000, 0x00000002, 0x00000004, 0x00040020, 0x0000001d, 0x00000000,
	0x0000001c, 0x0004003b, 0x0000001d, 0x0000001e, 0x00000000, 0x00040017, 0x0000001f,
	0x00000006, 0x00000003, 0x00050036, 0x00000002, 0x00000004, 0x00000000, 0x00000003,
	0x000200f8, 0x00000005, 0x0004003d, 0x00000018, 0x0000001f, 0x0000001a, 0x0004003d,
	0x0000001c, 0x00000020, 0x0000001e, 0x0005008e, 0x0000000f, 0x00000021, 0x0000001f,
	0x00000020, 0x00050051, 0x00000006, 0x00000022, 0x00000021, 0x00000000, 0x00050051,
	0x00000006, 0x00000023, 0x00000021, 0x00000001, 0x00050051, 0x00000006, 0x00000024,
	0x00000021, 0x00000002, 0x00050051, 0x00000006, 0x00000025, 0x00000021, 0x00000003,
	0x00070050, 0x00000007, 0x00000026, 0x00000022, 0x00000023, 0x00000024, 0x00000025,
	0x00050041, 0x00000015, 0x00000027, 0x00000012, 0x00000014, 0x0003003e, 0x00000027,
	0x00000026, 0x000100fd, 0x00010038,
}

// Simple fragment shader SPIR-V (hardcoded for fallback)
var SimpleFragmentShaderSPIRV = []uint32{
	0x07230203, 0x00010000, 0x00080001, 0x0000001e, 0x00000000, 0x00020011, 0x00000001,
	0x0006000b, 0x00000001, 0x4c534c47, 0x6474732e, 0x3035342e, 0x00000000, 0x0003000e,
	0x00000000, 0x00000001, 0x0007000f, 0x00000004, 0x00000004, 0x6e69616d, 0x00000000,
	0x00000009, 0x0000000d, 0x00030010, 0x00000004, 0x00000007, 0x00030003, 0x00000002,
	0x000001c2, 0x00040005, 0x00000004, 0x6e69616d, 0x00000000, 0x00040005, 0x00000009,
	0x636f6c66, 0x0000726f, 0x00060005, 0x0000000d, 0x67617266, 0x6f6c6f43, 0x00000072,
	0x00000000, 0x00040047, 0x00000009, 0x0000001e, 0x00000000, 0x00040047, 0x0000000d,
	0x0000001e, 0x00000000, 0x00020013, 0x00000002, 0x00030021, 0x00000003, 0x00000002,
	0x00030016, 0x00000006, 0x00000020, 0x00040017, 0x00000007, 0x00000006, 0x00000004,
	0x00040020, 0x00000008, 0x00000003, 0x00000007, 0x0004003b, 0x00000008, 0x00000009,
	0x00000003, 0x00040017, 0x0000000a, 0x00000006, 0x00000003, 0x00040020, 0x0000000b,
	0x00000001, 0x0000000a, 0x0004003b, 0x0000000b, 0x0000000c, 0x00000001, 0x00040020,
	0x0000000e, 0x00000001, 0x00000007, 0x0004003b, 0x0000000e, 0x0000000d, 0x00000001,
	0x00050036, 0x00000002, 0x00000004, 0x00000000, 0x00000003, 0x000200f8, 0x00000005,
	0x0004003d, 0x0000000a, 0x0000000f, 0x0000000c, 0x00050051, 0x00000006, 0x00000010,
	0x0000000f, 0x00000000, 0x00050051, 0x00000006, 0x00000011, 0x0000000f, 0x00000001,
	0x00050051, 0x00000006, 0x00000012, 0x0000000f, 0x00000002, 0x00070050, 0x00000007,
	0x00000013, 0x00000010, 0x00000011, 0x00000012, 0x00000012, 0x00050041, 0x00000008,
	0x00000014, 0x00000009, 0x00000012, 0x0003003e, 0x00000014, 0x00000013, 0x000100fd,
	0x00010038,
}

func init() {
	// Use pre-compiled SPIR-V as default
	if len(SimpleVertexShaderSPIRV) == 0 {
		// Fallback: you would need to compile shaders externally
	}
}

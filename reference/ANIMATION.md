# FCPXML Animation Parameter Structure

## Overview

This document captures the real-world implementation details for FCPXML animation parameters, specifically transform animations. These details differ significantly from what the DTD suggests is valid and represent the actual structure Final Cut Pro expects.

## Key Discoveries

### Parameter Hierarchy

FCPXML uses a nested parameter structure for transforms, not flat parameters:

**Incorrect (DTD-valid but FCP-incompatible):**
```xml
<param name="Position" value="0 0">
    <keyframeAnimation>
        <keyframe time="0s" value="0 0"/>
        <keyframe time="2s" value="640 0"/>
    </keyframeAnimation>
</param>
```

**Correct (FCP-compatible):**
```xml
<param name="position">
    <param name="X" key="1">
        <keyframeAnimation>
            <keyframe time="3600s" value="-266.667"/>
            <keyframe time="10803300/3000s" value="0"/>
        </keyframeAnimation>
    </param>
    <param name="Y" key="2">
        <keyframeAnimation>
            <keyframe time="3600s" value="0" curve="linear"/>
        </keyframeAnimation>
    </param>
</param>
```

## Critical Implementation Details

### 1. Case Sensitivity
- Parameter names are case-sensitive
- Use `"position"` not `"Position"`
- Use `"X"` and `"Y"` for axis components

### 2. Key Attributes
- Each parameter component requires a `key` attribute
- Standard keys: `key="1"` for X-axis, `key="2"` for Y-axis
- Keys provide unique identification for parameter components

### 3. Timing Context
- Animation timing is **relative to media start time**, not absolute project timeline
- If video starts at `start="3600s"`, keyframes begin at `time="3600s"`
- End time calculations: `start_time + duration` (e.g., `3600s + 2.4s = 10803300/3000s`)
- **CRITICAL**: For asset-clip animations (video layer), timing is **relative to clip offset**, not media start time
- Asset-clip keyframes start at `time="0s"` regardless of video element timing

### 4. Component-Level Animation
- Each axis (X, Y) is animated independently
- Allows different interpolation curves per axis
- Y-axis can remain static while X-axis animates

### 5. Value Precision
- Use decimal values for sub-pixel positioning
- Example: `-266.667` for precise positioning calculations
- Maintain precision for smooth animations

## Complete Transform Animation Examples

### Video Element Animation (Overlay Layer)
```xml
<asset-clip ref="r2" offset="0s" duration="3980/600s">
    <video ref="r4" lane="1" offset="0s" start="3600s" duration="20400/3000s">
        <adjust-transform>
            <param name="position">
                <param name="X" key="1">
                    <keyframeAnimation>
                        <keyframe time="3600s" value="-266.667"/>
                        <keyframe time="10803300/3000s" value="0"/>
                    </keyframeAnimation>
                </param>
                <param name="Y" key="2">
                    <keyframeAnimation>
                        <keyframe time="3600s" value="0" curve="linear"/>
                    </keyframeAnimation>
                </param>
            </param>
        </adjust-transform>
    </video>
</asset-clip>
```

### Asset-Clip Animation (Video Layer)
```xml
<asset-clip ref="r2" offset="0s" duration="3980/600s">
    <adjust-transform>
        <param name="position">
            <param name="X" key="1">
                <keyframeAnimation>
                    <keyframe time="0s" value="0"/>
                    <keyframe time="2200/3000s" value="133.333"/>
                </keyframeAnimation>
            </param>
            <param name="Y" key="2">
                <keyframeAnimation>
                    <keyframe time="0s" value="0" curve="linear"/>
                </keyframeAnimation>
            </param>
        </param>
    </adjust-transform>
    <video ref="r4" lane="1" offset="0s" start="3600s" duration="20400/3000s">
        <!-- video element animations use different timing -->
    </video>
</asset-clip>
```

## Animation Types

### Position Animation
- Container parameter: `"position"`
- Components: `"X"` (key="1"), `"Y"` (key="2")
- Values: Pixel coordinates relative to center

### Scale Animation
- Container parameter: `"scale"`
- Components: `"X"` (key="1"), `"Y"` (key="2")
- Values: Scale factors (1.0 = 100%)

### Rotation Animation
- Parameter: `"rotation"`
- Values: Degrees (360 = full rotation)

## Time Calculation

### Frame-Accurate Timing
- Use rational time representation: `numerator/denominator + s`
- Example: `10803300/3000s` = 3601.1 seconds
- Maintains frame accuracy for smooth playback

### Duration Calculations
```
End Time = Start Time + Animation Duration
3600s + 2.4s = 3602.4s = 10807200/3000s
```

## Common Pitfalls

### 1. DTD vs Reality
- DTD validation may pass with incorrect parameter structure
- Final Cut Pro expects specific nested hierarchy
- Always test imports in actual FCP application

### 2. Timing Mistakes
- Using absolute project time instead of relative media time
- Forgetting to account for media start offset
- Incorrect rational time calculations
- **Mixing timing contexts**: Asset-clip animations use clip-relative timing (start at 0s), video element animations use media-relative timing

### 3. Parameter Naming
- Case sensitivity errors (`Position` vs `position`)
- Missing key attributes on parameter components
- Incorrect component names (`x` vs `X`)

### 4. Animation Layer Confusion
- Asset-clip transforms affect the entire video clip
- Video element transforms affect overlay layers (generators, effects)
- Duration calculations differ between layers
- Coordinate systems may vary between animation contexts

## Best Practices

1. **Always use nested parameter structure** for transforms
2. **Include key attributes** for all parameter components
3. **Calculate timing relative to media start time**
4. **Test animations in Final Cut Pro** to verify behavior
5. **Use precise decimal values** for smooth motion
6. **Specify interpolation curves** when needed (`curve="linear"`)

## Validation Strategy

1. **DTD Validation**: Ensures XML structure validity
2. **FCP Import Test**: Confirms actual compatibility
3. **Animation Playback**: Verifies expected behavior

**Note**: DTD validation alone is insufficient - Final Cut Pro compatibility requires testing actual imports and playback behavior.

## Related Elements

- `adjust-transform`: Container for transform parameters
- `keyframeAnimation`: Container for keyframes
- `keyframe`: Individual animation points with time/value/curve
- `param`: Parameter containers and components

This structure applies to all transform-based animations in FCPXML, including position, scale, rotation, and advanced transform properties.
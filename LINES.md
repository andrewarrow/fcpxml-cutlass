# FCPXML Table Line Positioning Guide

## Overview

This document captures the critical insights learned while implementing table grid lines in FCPXML format. Creating proper edge-to-edge table lines requires specific positioning techniques that are not intuitive from the FCPXML specification alone.

## Key Insights

### 1. Position-Based Transforms vs. Center Parameters

**WRONG Approach (doesn't work):**
```xml
<video ref="r2" lane="1">
    <param name="Center" value="0.5 0.150"/>
    <adjust-transform position="0 8.33333" scale="1 0.0394"/>
</video>
```

**CORRECT Approach (works):**
```xml
<video ref="r2" lane="1">
    <!-- NO Center parameter -->
    <adjust-transform position="0 -100.0" scale="1 0.0394"/>
</video>
```

**Why:** Lines must use position-based transforms from the default center (0.5, 0.5) rather than changing the Center parameter. The Center parameter doesn't create the desired edge-to-edge effect.

### 2. Line Thickness and Scaling

**Horizontal Lines (span full width):**
- Scale: `scale="1 0.0394"`
- Position format: `position="0 Y_OFFSET"`

**Vertical Lines (span full height):**
- Scale: `scale="0.0394 1"`  
- Position format: `position="X_OFFSET 0"`

**Critical:** Both horizontal and vertical lines use the same thickness value (0.0394), but applied to different axes.

### 3. Exact Positioning Values for Edge-to-Edge Coverage

**Don't calculate evenly distributed positions.** Use these exact values from working `wide.fcpxml`:

**Horizontal Line Positions (Y-axis):**
```go
horizontalPositionOffsets := []float64{-100, -46.5928, 48.0135, 100}
```

**Vertical Line Positions (X-axis):**
```go
verticalPositionOffsets := []float64{-150, -73.3652, 73.3319, 150}
```

**Why these values work:** They ensure lines extend to the exact edges of the Final Cut Pro frame boundaries. Mathematical distribution doesn't account for frame edge positioning.

### 4. Nested Video Structure

**Main spine video** contains all lines as **nested videos with lanes:**

```xml
<spine>
    <video ref="r2" offset="0s" name="Table Grid Base">
        <!-- Main video params without Center -->
        <adjust-transform scale="1 0.0395"/>
        
        <!-- Horizontal lines nested with lanes -->
        <video ref="r2" lane="1" name="Horizontal Line 1">
            <adjust-transform position="0 -100" scale="1 0.0394"/>
        </video>
        <video ref="r2" lane="2" name="Horizontal Line 2">
            <adjust-transform position="0 -46.5928" scale="1 0.0394"/>
        </video>
        
        <!-- Vertical lines nested with lanes -->
        <video ref="r2" lane="5" name="Vertical Line 1">
            <adjust-transform position="-150 0" scale="0.0394 1"/>
        </video>
        <!-- etc... -->
    </video>
</spine>
```

### 5. Line Count Formula

For a table with **N data rows** and **M columns:**
- **Horizontal lines needed:** N + 1 (to create N rows)
- **Vertical lines needed:** M + 1 (to create M columns)

**Example:** 2 data rows + 1 header = 3 rows total
- Need **4 horizontal lines** (top, middle1, middle2, bottom)
- Need **4 vertical lines** for 3 columns (left, middle1, middle2, right)

### 6. Parameter Structure

**Essential parameters for all lines:**
```xml
<param name="Drop Shadow Opacity" key="9999/988455508/1/208/211" value="0.7426"/>
<param name="Feather" key="9999/988455508/988455699/2/353/102" value="3"/>
<param name="Fill Color" key="9999/988455508/988455699/2/353/113/111" value="1.0817 -0.0799793 -0.145856"/>
<param name="Falloff" key="9999/988455508/988455699/2/353/158" value="-2"/>
<param name="Shape" key="9999/988461322/100/988461395/2/100" value="4 (Rectangle)"/>
<param name="Outline" key="9999/988461322/100/988464485/2/100" value="0"/>
<param name="Outline Width" key="9999/988461322/100/988467855/2/100" value="0.338788"/>
<param name="Corners" key="9999/988461322/100/988469428/2/100" value="1 (Square)"/>
```

**Do NOT include Center parameter** in nested line videos.

### 7. Lane Numbering

**Sequential lane numbering** for nested elements:
- Lane 1-4: Horizontal lines (top to bottom)
- Lane 5-8: Vertical lines (left to right)

Each nested video must have a unique lane number to stack properly.

### 8. Resource Dependencies

**Required resources:**
```xml
<resources>
    <format id="r1" name="FFVideoFormat1080p2997" frameDuration="1001/30000s" width="1920" height="1080"/>
    <effect id="r2" name="Shapes" uid=".../Generators.localized/Elements.localized/Shapes.localized/Shapes.motn"/>
</resources>
```

All line videos reference the "Shapes" effect (r2).

## Common Mistakes

### 1. Using Center Parameters
❌ **Don't do this:**
```xml
<param name="Center" value="0.367 0.5"/>
```

✅ **Do this instead:**
```xml
<adjust-transform position="-73.3652 0"/>
```

### 2. Calculating Even Distribution
❌ **Don't calculate positions like this:**
```go
yOffset := -totalHeight/2 + totalHeight*float64(i)/float64(maxVisibleRows)
```

✅ **Use exact hardcoded values:**
```go
horizontalPositionOffsets := []float64{-100, -46.5928, 48.0135, 100}
```

### 3. Wrong Line Count
❌ **Wrong:** Using maxRows instead of maxRows+1 lines

✅ **Correct:** Always use N+1 lines to create N cells

### 4. Inconsistent Scaling
❌ **Wrong:** Using different thickness values for horizontal vs vertical

✅ **Correct:** Use 0.0394 for both, but on different axes

## Testing and Verification

To verify lines extend to edges:
1. Generate FCPXML file using exact positioning values
2. Import into Final Cut Pro
3. Check that lines reach the exact edges of the frame preview
4. Verify 4 horizontal and 4 vertical lines are visible
5. Confirm lines create properly sized cells for text content

## Implementation Notes

The exact positioning values (-100, -46.5928, etc.) were reverse-engineered from a working `wide.fcpxml` file that correctly displayed edge-to-edge lines. These values appear to be calibrated for standard 1920x1080 Final Cut Pro timeline positioning.

For different resolutions or aspect ratios, these values may need adjustment, but the underlying principles (position-based transforms, no Center parameters, proper line counts) remain the same.
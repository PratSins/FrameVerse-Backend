package gemini

const CommonPromptSuffix = `This is a strict pixel-aligned edit of the source video: keep the same pose, motion, timing, clothing colors, and background.

The camera must not change — no zoom, no crop, no recentering, and no change to the field of view.

The person's face and body must stay at exactly the same position and size in the frame as the source.

Eyes, nose, and mouth must remain at the same screen coordinates in every frame.

Match the facial expression exactly, frame by frame.

Preserve the exact degree of mouth openness at every moment — if the mouth is slightly open and still, keep it slightly open and still; do not close it, and do not add talking or any mouth movement that is not in the source.

Mirror blinks, gaze direction, and eyebrow position at the same moments as the source.

Change only the visual style, nothing about the geometry, composition, or performance.`

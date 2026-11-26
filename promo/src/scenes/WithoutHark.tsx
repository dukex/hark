import {
  AbsoluteFill,
  Easing,
  Interactive,
  interpolate,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";

export const WithoutHark = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  return (
    <AbsoluteFill
      style={{
        justifyContent: "center",
        padding: "0 160px",
        gap: 44,
        fontFamily: "Inter",
        fontSize: 78,
        color: "#FFFFFF",
      }}
    >
      <Interactive.Div
        name="Label"
        style={{
          fontWeight: 700,
          color: "#9E9E9E",
          marginBottom: 24,
          opacity: interpolate(frame, [0, 0.5 * fps], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
        }}
      >
        Without hark
      </Interactive.Div>
      <Interactive.Div
        name="10:05 row"
        style={{
          display: "flex",
          gap: 48,
          opacity: interpolate(frame, [1 * fps, 1.5 * fps], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
          translate: interpolate(
            frame,
            [1 * fps, 1.6 * fps],
            ["40px 0px", "0px 0px"],
            {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
              easing: Easing.bezier(0.16, 1, 0.3, 1),
            },
          ),
        }}
      >
        <span style={{ fontFamily: "JetBrains Mono", color: "#9E9E9E" }}>
          10:05
        </span>
        <span>Carol requests changes on #15</span>
      </Interactive.Div>
      <Interactive.Div
        name="10:12 row"
        style={{
          display: "flex",
          gap: 48,
          opacity: interpolate(frame, [2.2 * fps, 2.7 * fps], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
          translate: interpolate(
            frame,
            [2.2 * fps, 2.8 * fps],
            ["40px 0px", "0px 0px"],
            {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
              easing: Easing.bezier(0.16, 1, 0.3, 1),
            },
          ),
        }}
      >
        <span style={{ fontFamily: "JetBrains Mono", color: "#9E9E9E" }}>
          10:12
        </span>
        <span>CI fails on main</span>
      </Interactive.Div>
      <Interactive.Div
        name="10:31 row"
        style={{
          display: "flex",
          gap: 48,
          opacity: interpolate(frame, [3.4 * fps, 3.9 * fps], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
          translate: interpolate(
            frame,
            [3.4 * fps, 4 * fps],
            ["40px 0px", "0px 0px"],
            {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
              easing: Easing.bezier(0.16, 1, 0.3, 1),
            },
          ),
        }}
      >
        <span style={{ fontFamily: "JetBrains Mono", color: "#9E9E9E" }}>
          10:31
        </span>
        <span>Ana asks for you in #deploys</span>
      </Interactive.Div>
      <Interactive.Div
        name="11:40 row"
        style={{
          display: "flex",
          gap: 48,
          color: "#F44336",
          fontWeight: 700,
          opacity: interpolate(frame, [5.4 * fps, 5.9 * fps], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
          translate: interpolate(
            frame,
            [5.4 * fps, 6 * fps],
            ["40px 0px", "0px 0px"],
            {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
              easing: Easing.bezier(0.16, 1, 0.3, 1),
            },
          ),
        }}
      >
        <span style={{ fontFamily: "JetBrains Mono", fontWeight: 400 }}>
          11:40
        </span>
        <span>You find all three, 95 min later</span>
      </Interactive.Div>
    </AbsoluteFill>
  );
};

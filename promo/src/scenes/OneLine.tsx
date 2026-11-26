import {
  AbsoluteFill,
  Interactive,
  interpolate,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";

export const OneLine = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  return (
    <AbsoluteFill
      style={{ justifyContent: "center", padding: "0 140px", gap: 64 }}
    >
      <Interactive.Div
        name="Headline"
        style={{
          fontFamily: "Inter",
          fontWeight: 700,
          fontSize: 150,
          letterSpacing: "-0.03em",
          color: "#FFFFFF",
          opacity: interpolate(frame, [0, 0.5 * fps], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
        }}
      >
        One line per event
      </Interactive.Div>
      <div
        style={{
          display: "flex",
          flexDirection: "column",
          gap: 20,
          padding: "40px 48px",
          backgroundColor: "#1A1A1A",
          border: "2px solid #333333",
          borderRadius: 12,
          fontFamily: "JetBrains Mono",
          fontSize: 44,
          color: "#FFFFFF",
          whiteSpace: "pre",
        }}
      >
        <Interactive.Div name="Prompt" style={{ color: "#9E9E9E" }}>
          $ hark github
        </Interactive.Div>
        <Interactive.Div
          name="Line 1"
          style={{
            opacity: interpolate(frame, [1 * fps, 1.2 * fps], [0, 1], {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
            }),
          }}
        >
          10:05 github pr.changes_requested acme/checkout#15 @carol
        </Interactive.Div>
        <Interactive.Div
          name="Line 2"
          style={{
            opacity: interpolate(frame, [2 * fps, 2.2 * fps], [0, 1], {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
            }),
          }}
        >
          10:12 github ci.failed acme/checkout &quot;CI failed on main&quot;
        </Interactive.Div>
        <Interactive.Div
          name="Line 3"
          style={{
            opacity: interpolate(frame, [3 * fps, 3.2 * fps], [0, 1], {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
            }),
          }}
        >
          10:20 github pr.approved acme/lib#40 @frank
        </Interactive.Div>
        <Interactive.Div
          name="Line 4"
          style={{
            opacity: interpolate(frame, [4 * fps, 4.2 * fps], [0, 1], {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
            }),
          }}
        >
          10:24 github pr.merged dukex/foo#12 @bob
        </Interactive.Div>
      </div>
    </AbsoluteFill>
  );
};

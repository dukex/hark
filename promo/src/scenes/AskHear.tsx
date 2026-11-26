import {
  AbsoluteFill,
  Easing,
  Interactive,
  interpolate,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";

export const AskHear = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  return (
    <AbsoluteFill
      style={{
        justifyContent: "center",
        alignItems: "center",
        padding: "0 140px",
        gap: 24,
        fontFamily: "Inter",
        textAlign: "center",
      }}
    >
      <Interactive.Div
        name="Ask"
        style={{
          fontWeight: 600,
          fontSize: 96,
          color: "#9E9E9E",
          opacity: interpolate(frame, [0, 0.5 * fps], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
        }}
      >
        MCP lets your agent ask.
      </Interactive.Div>
      <Interactive.Div
        name="Hear"
        style={{
          fontWeight: 700,
          fontSize: 150,
          letterSpacing: "-0.03em",
          color: "#FFFFFF",
          opacity: interpolate(frame, [1.2 * fps, 1.7 * fps], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
          scale: interpolate(frame, [1.2 * fps, 2 * fps], [0.92, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
            easing: Easing.bezier(0.16, 1, 0.3, 1),
            output: "perceptual-scale",
          }),
        }}
      >
        hark lets it hear.
      </Interactive.Div>
      <Interactive.Div
        name="Services"
        style={{
          display: "flex",
          flexWrap: "wrap",
          justifyContent: "center",
          gap: 24,
          marginTop: 72,
          fontWeight: 600,
          fontSize: 56,
          color: "#FFFFFF",
          opacity: interpolate(frame, [2.8 * fps, 3.4 * fps], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
          translate: interpolate(
            frame,
            [2.8 * fps, 3.5 * fps],
            ["0px 32px", "0px 0px"],
            {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
              easing: Easing.bezier(0.16, 1, 0.3, 1),
            },
          ),
        }}
      >
        {[
          "GitHub",
          "Slack",
          "Discord",
          "Jira",
          "Linear",
          "Datadog",
          "Todoist",
        ].map((service) => (
          <span
            key={service}
            style={{
              padding: "16px 40px",
              borderRadius: 9999,
              backgroundColor: "#1A1A1A",
              border: "2px solid #333333",
            }}
          >
            {service}
          </span>
        ))}
      </Interactive.Div>
    </AbsoluteFill>
  );
};

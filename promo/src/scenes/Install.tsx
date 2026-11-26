import {
  AbsoluteFill,
  Easing,
  Interactive,
  interpolate,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { Logo } from "../Logo";

export const Install = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  return (
    <AbsoluteFill
      style={{
        justifyContent: "center",
        alignItems: "center",
        gap: 64,
        fontFamily: "Inter",
      }}
    >
      <Interactive.Div
        name="Brand"
        style={{
          display: "flex",
          alignItems: "center",
          gap: 40,
          fontWeight: 700,
          fontSize: 150,
          letterSpacing: "-0.03em",
          color: "#FFFFFF",
          opacity: interpolate(frame, [0, 0.5 * fps], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
          scale: interpolate(frame, [0, 0.8 * fps], [0.9, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
            easing: Easing.bezier(0.16, 1, 0.3, 1),
            output: "perceptual-scale",
          }),
        }}
      >
        <Logo size={150} />
        hark
      </Interactive.Div>
      <Interactive.Div
        name="Install command"
        style={{
          padding: "28px 48px",
          borderRadius: 12,
          backgroundColor: "#1A1A1A",
          border: "3px solid #F2C94C",
          fontFamily: "JetBrains Mono",
          fontSize: 52,
          color: "#FFFFFF",
          whiteSpace: "pre",
          opacity: interpolate(frame, [0.9 * fps, 1.4 * fps], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
        }}
      >
        curl -fsSL https://hark.dukex.sh/install | sh
      </Interactive.Div>
      <Interactive.Div
        name="Footer"
        style={{
          fontSize: 78,
          color: "#9E9E9E",
          opacity: interpolate(frame, [1.8 * fps, 2.3 * fps], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
        }}
      >
        Free, for Linux and macOS
      </Interactive.Div>
    </AbsoluteFill>
  );
};

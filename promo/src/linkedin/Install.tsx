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
        gap: 56,
        padding: "0 80px",
        fontFamily: "Inter",
        textAlign: "center",
      }}
    >
      <Interactive.Div
        name="Brand"
        style={{
          display: "flex",
          alignItems: "center",
          gap: 32,
          fontWeight: 700,
          fontSize: 140,
          letterSpacing: "-0.03em",
          color: "#FFFFFF",
          scale: interpolate(frame, [0, 0.8 * fps], [0.9, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
            easing: Easing.bezier(0.16, 1, 0.3, 1),
            output: "perceptual-scale",
          }),
        }}
      >
        <Logo size={140} />
        hark
      </Interactive.Div>
      <Interactive.Div
        name="Site"
        style={{
          fontWeight: 700,
          fontSize: 84,
          color: "#F2C94C",
          opacity: interpolate(frame, [0.6 * fps, 1 * fps], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
        }}
      >
        hark.dukex.sh
      </Interactive.Div>
      <Interactive.Div
        name="Install command"
        style={{
          padding: "22px 28px",
          borderRadius: 12,
          backgroundColor: "#1A1A1A",
          border: "2px solid #333333",
          fontFamily: "JetBrains Mono",
          fontSize: 30,
          color: "#FFFFFF",
          whiteSpace: "pre",
          opacity: interpolate(frame, [1.1 * fps, 1.5 * fps], [0, 1], {
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
          fontSize: 44,
          color: "#9E9E9E",
          opacity: interpolate(frame, [1.6 * fps, 2 * fps], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
        }}
      >
        Free and open source, for Linux and macOS
      </Interactive.Div>
    </AbsoluteFill>
  );
};

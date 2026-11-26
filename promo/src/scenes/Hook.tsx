import {
  AbsoluteFill,
  Easing,
  Interactive,
  interpolate,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { Logo } from "../Logo";

export const Hook = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  return (
    <AbsoluteFill
      style={{ justifyContent: "center", padding: "0 160px", gap: 56 }}
    >
      <Interactive.Div
        name="Brand"
        style={{
          display: "flex",
          alignItems: "center",
          gap: 24,
          fontFamily: "Inter",
          fontWeight: 700,
          fontSize: 64,
          color: "#FFFFFF",
          opacity: interpolate(frame, [0, 0.5 * fps], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
        }}
      >
        <Logo size={72} />
        hark
      </Interactive.Div>
      <Interactive.Div
        name="Headline"
        style={{
          fontFamily: "Inter",
          fontWeight: 700,
          fontSize: 150,
          lineHeight: 1.05,
          letterSpacing: "-0.03em",
          color: "#FFFFFF",
          opacity: interpolate(frame, [0.3 * fps, 1 * fps], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
          translate: interpolate(
            frame,
            [0.3 * fps, 1.2 * fps],
            ["0px 48px", "0px 0px"],
            {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
              easing: Easing.bezier(0.16, 1, 0.3, 1),
            },
          ),
        }}
      >
        Keep your coding agent in the loop
      </Interactive.Div>
      <Interactive.Div
        name="Subhead"
        style={{
          fontFamily: "Inter",
          fontWeight: 400,
          fontSize: 78,
          lineHeight: 1.2,
          color: "#9E9E9E",
          opacity: interpolate(frame, [1.6 * fps, 2.2 * fps], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
        }}
      >
        Reviews, CI runs, pings and alerts reach it while it works.
      </Interactive.Div>
    </AbsoluteFill>
  );
};

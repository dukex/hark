import {
  AbsoluteFill,
  Easing,
  Interactive,
  interpolate,
  interpolateColors,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { Logo } from "../Logo";
import { Typed } from "../Typed";

type Props = {
  readonly provider: string;
  readonly event: string;
  readonly command: string;
  readonly kind: string;
  readonly detail: string;
  readonly reply: string;
};

const clamp = {
  extrapolateLeft: "clamp",
  extrapolateRight: "clamp",
} as const;

export const EventFlow = ({
  provider,
  event,
  command,
  kind,
  detail,
  reply,
}: Props) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const printed = Math.round(
    interpolate(
      frame,
      [2 * fps, 2.6 * fps],
      [0, kind.length + detail.length],
      clamp,
    ),
  );
  const typed = Math.round(
    interpolate(frame, [5.2 * fps, 8 * fps], [0, reply.length], clamp),
  );

  return (
    <AbsoluteFill>
      <Interactive.Div
        name="Camera"
        style={{
          position: "absolute",
          inset: 0,
          transformOrigin: "1500px 540px",
          scale: interpolate(frame, [4.4 * fps, 5.2 * fps], [1, 1.6], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
            easing: Easing.bezier(0.65, 0, 0.35, 1),
            output: "perceptual-scale",
          }),
          translate: interpolate(
            frame,
            [4.4 * fps, 5.2 * fps],
            ["0px 0px", "-540px 0px"],
            {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
              easing: Easing.bezier(0.65, 0, 0.35, 1),
            },
          ),
        }}
      >
        <svg
          width={1920}
          height={1080}
          style={{ position: "absolute", inset: 0 }}
        >
          <line
            x1={440}
            y1={540}
            x2={600}
            y2={540}
            stroke="#333333"
            strokeWidth={4}
            strokeDasharray="12 10"
          />
          <line
            x1={1020}
            y1={540}
            x2={1180}
            y2={540}
            stroke="#333333"
            strokeWidth={4}
            strokeDasharray="12 10"
          />
        </svg>

        <div
          style={{
            position: "absolute",
            left: 100,
            top: 400,
            width: 340,
            height: 280,
            display: "flex",
            flexDirection: "column",
            gap: 20,
            padding: 32,
            backgroundColor: "#1A1A1A",
            border: "3px solid",
            borderColor: interpolateColors(
              frame,
              [0.6 * fps, 1 * fps, 2 * fps],
              ["#333333", "#FFFFFF", "#333333"],
            ),
            borderRadius: 16,
            fontFamily: "Inter",
          }}
        >
          <span style={{ fontSize: 44, fontWeight: 700, color: "#FFFFFF" }}>
            {provider}
          </span>
          <span
            style={{
              fontSize: 32,
              lineHeight: 1.3,
              color: "#9E9E9E",
              opacity: interpolate(frame, [0.6 * fps, 1 * fps], [0, 1], clamp),
            }}
          >
            {event}
          </span>
        </div>

        <div
          style={{
            position: "absolute",
            left: 600,
            top: 420,
            width: 420,
            height: 240,
            display: "flex",
            flexDirection: "column",
            gap: 16,
            padding: 24,
            backgroundColor: "#1A1A1A",
            border: "3px solid",
            borderColor: interpolateColors(
              frame,
              [2 * fps, 2.4 * fps, 3.4 * fps],
              ["#333333", "#FFFFFF", "#333333"],
            ),
            borderRadius: 16,
            fontFamily: "JetBrains Mono",
            fontSize: 30,
          }}
        >
          <span
            style={{
              display: "flex",
              alignItems: "center",
              gap: 16,
              color: "#9E9E9E",
            }}
          >
            <Logo size={40} />$ {command}
          </span>
          <span style={{ fontWeight: 700, color: "#FFFFFF" }}>
            <Typed text={kind} count={printed} />
          </span>
          <span style={{ color: "#9E9E9E" }}>
            <Typed text={detail} count={printed - kind.length} />
          </span>
        </div>

        <div
          style={{
            position: "absolute",
            left: 1180,
            top: 220,
            width: 640,
            height: 640,
            display: "flex",
            flexDirection: "column",
            backgroundColor: "#1A1A1A",
            border: "3px solid",
            borderColor: interpolateColors(
              frame,
              [3.6 * fps, 4 * fps, 5 * fps],
              ["#333333", "#FFFFFF", "#333333"],
            ),
            borderRadius: 16,
            boxShadow: "0 8px 16px rgba(0, 0, 0, 0.8)",
            overflow: "hidden",
            fontFamily: "Inter",
          }}
        >
          <div
            style={{
              display: "flex",
              alignItems: "center",
              height: 56,
              padding: "0 20px",
              borderBottom: "2px solid #333333",
            }}
          >
            <div style={{ display: "flex", gap: 10 }}>
              {[0, 1, 2].map((dot) => (
                <span
                  key={dot}
                  style={{
                    width: 16,
                    height: 16,
                    borderRadius: 9999,
                    backgroundColor: "#333333",
                  }}
                />
              ))}
            </div>
            <span
              style={{
                flex: 1,
                textAlign: "center",
                marginRight: 68,
                fontSize: 26,
                fontWeight: 600,
                color: "#9E9E9E",
              }}
            >
              Claude
            </span>
          </div>
          <div
            style={{
              display: "flex",
              flexDirection: "column",
              gap: 22,
              padding: 32,
            }}
          >
            <span
              style={{
                alignSelf: "flex-end",
                maxWidth: 440,
                padding: "14px 22px",
                borderRadius: 16,
                backgroundColor: "#262626",
                fontSize: 28,
                lineHeight: 1.3,
                color: "#FFFFFF",
              }}
            >
              Move the refund flow to the new payments client
            </span>
            <span
              style={{
                fontFamily: "JetBrains Mono",
                fontSize: 24,
                color: "#9E9E9E",
              }}
            >
              Editing refund.go
            </span>
            <span
              style={{
                alignSelf: "flex-start",
                padding: "10px 18px",
                borderRadius: 12,
                border: "2px solid #333333",
                backgroundColor: "#262626",
                fontFamily: "JetBrains Mono",
                fontSize: 24,
                color: "#FFFFFF",
                opacity: interpolate(frame, [3.6 * fps, 3.9 * fps], [0, 1], clamp),
              }}
            >
              {command} · {kind}
            </span>
            <span
              style={{
                display: "flex",
                gap: 18,
                fontSize: 42,
                lineHeight: 1.25,
                color: "#FFFFFF",
                opacity: interpolate(frame, [5 * fps, 5.2 * fps], [0, 1], clamp),
              }}
            >
              <span
                style={{
                  flexShrink: 0,
                  width: 14,
                  height: 14,
                  marginTop: 20,
                  borderRadius: 9999,
                  backgroundColor: "#FFFFFF",
                }}
              />
              <span>
                <Typed text={reply} count={typed} />
              </span>
            </span>
          </div>
        </div>

        <Interactive.Div
          name="Event pulse"
          style={{
            position: "absolute",
            top: 528,
            width: 24,
            height: 24,
            borderRadius: 9999,
            backgroundColor: "#FFFFFF",
            boxShadow: "0 0 24px 8px rgba(255, 255, 255, 0.45)",
            left: interpolate(
              frame,
              [1.2 * fps, 2 * fps, 2.8 * fps, 3.6 * fps],
              [430, 590, 1010, 1170],
              {
                extrapolateLeft: "clamp",
                extrapolateRight: "clamp",
              },
            ),
            opacity: interpolate(
              frame,
              [
                1.2 * fps,
                1.3 * fps,
                1.9 * fps,
                2 * fps,
                2.8 * fps,
                2.9 * fps,
                3.5 * fps,
                3.6 * fps,
              ],
              [0, 1, 1, 0, 0, 1, 1, 0],
              {
                extrapolateLeft: "clamp",
                extrapolateRight: "clamp",
              },
            ),
          }}
        />
      </Interactive.Div>
    </AbsoluteFill>
  );
};

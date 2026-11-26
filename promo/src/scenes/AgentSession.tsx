import {
  AbsoluteFill,
  Easing,
  Interactive,
  interpolate,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";

type Props = {
  readonly source: string;
  readonly kind: string;
  readonly detail: string;
  readonly reply: string;
};

export const AgentSession = ({ source, kind, detail, reply }: Props) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const typed = Math.round(
    interpolate(frame, [3 * fps, 6.5 * fps], [0, reply.length], {
      extrapolateLeft: "clamp",
      extrapolateRight: "clamp",
    }),
  );

  return (
    <AbsoluteFill style={{ padding: "140px 140px" }}>
      <div
        style={{
          flex: 1,
          display: "flex",
          flexDirection: "column",
          backgroundColor: "#1A1A1A",
          border: "2px solid #333333",
          borderRadius: 12,
          boxShadow: "0 8px 16px rgba(0, 0, 0, 0.8)",
          overflow: "hidden",
        }}
      >
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            padding: "24px 48px",
            borderBottom: "2px solid #333333",
            fontFamily: "JetBrains Mono",
            fontSize: 40,
            color: "#9E9E9E",
          }}
        >
          <span>~/code/checkout · editing refund.go</span>
          <span
            style={{
              display: "flex",
              alignItems: "center",
              gap: 16,
              padding: "8px 24px",
              borderRadius: 9999,
              backgroundColor: "rgba(76, 175, 80, 0.12)",
              color: "#4CAF50",
              fontFamily: "Inter",
              fontWeight: 600,
            }}
          >
            <span
              style={{
                width: 16,
                height: 16,
                borderRadius: 9999,
                backgroundColor: "#4CAF50",
              }}
            />
            Monitors running
          </span>
        </div>
        <div
          style={{
            flex: 1,
            display: "flex",
            flexDirection: "column",
            justifyContent: "center",
            gap: 56,
            padding: "0 56px",
          }}
        >
          <Interactive.Div
            name="Event"
            style={{
              display: "flex",
              flexDirection: "column",
              gap: 12,
              paddingLeft: 32,
              borderLeft: "6px solid #9E9E9E",
              fontFamily: "JetBrains Mono",
              opacity: interpolate(frame, [0.8 * fps, 1.3 * fps], [0, 1], {
                extrapolateLeft: "clamp",
                extrapolateRight: "clamp",
              }),
              translate: interpolate(
                frame,
                [0.8 * fps, 1.4 * fps],
                ["0px 32px", "0px 0px"],
                {
                  extrapolateLeft: "clamp",
                  extrapolateRight: "clamp",
                  easing: Easing.bezier(0.16, 1, 0.3, 1),
                },
              ),
            }}
          >
            <span style={{ fontSize: 40, color: "#9E9E9E" }}>{source}</span>
            <span style={{ fontSize: 64, fontWeight: 700, color: "#FFFFFF" }}>
              {kind}
            </span>
            <span style={{ fontSize: 48, color: "#9E9E9E" }}>{detail}</span>
          </Interactive.Div>
          <Interactive.Div
            name="Reply"
            style={{
              display: "flex",
              gap: 32,
              fontFamily: "Inter",
              fontSize: 78,
              lineHeight: 1.2,
              color: "#FFFFFF",
              opacity: interpolate(frame, [2.6 * fps, 3 * fps], [0, 1], {
                extrapolateLeft: "clamp",
                extrapolateRight: "clamp",
              }),
            }}
          >
            <span
              style={{
                flexShrink: 0,
                width: 24,
                height: 24,
                marginTop: 34,
                borderRadius: 9999,
                backgroundColor: "#FFFFFF",
              }}
            />
            <span>
              {reply.slice(0, typed)}
              <span style={{ opacity: 0 }}>{reply.slice(typed)}</span>
            </span>
          </Interactive.Div>
        </div>
      </div>
    </AbsoluteFill>
  );
};

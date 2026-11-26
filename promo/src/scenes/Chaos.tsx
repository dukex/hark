import {
  AbsoluteFill,
  Easing,
  Interactive,
  interpolate,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";

type WindowProps = {
  readonly name: string;
  readonly lines: readonly string[];
  readonly unread: number;
  readonly left: number;
  readonly top: number;
  readonly opensAt: number;
};

const AppWindow = ({
  name,
  lines,
  unread,
  left,
  top,
  opensAt,
}: WindowProps) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const open = interpolate(
    frame,
    [opensAt * fps, (opensAt + 0.35) * fps],
    [0, 1],
    {
      extrapolateLeft: "clamp",
      extrapolateRight: "clamp",
      easing: Easing.bezier(0.16, 1, 0.3, 1),
    },
  );

  return (
    <div
      style={{
        position: "absolute",
        left,
        top,
        width: 620,
        height: 340,
        display: "flex",
        flexDirection: "column",
        backgroundColor: "#1A1A1A",
        border: "2px solid #333333",
        borderRadius: 14,
        boxShadow: "0 8px 16px rgba(0, 0, 0, 0.8)",
        overflow: "hidden",
        fontFamily: "Inter",
        opacity: open,
        scale: 0.85 + 0.15 * open,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 14,
          height: 56,
          padding: "0 20px",
          borderBottom: "2px solid #333333",
          backgroundColor: "#262626",
        }}
      >
        {[0, 1, 2].map((dot) => (
          <span
            key={dot}
            style={{
              width: 14,
              height: 14,
              borderRadius: 9999,
              backgroundColor: "#333333",
            }}
          />
        ))}
        <span
          style={{
            flex: 1,
            marginLeft: 12,
            fontSize: 28,
            fontWeight: 700,
            color: "#FFFFFF",
          }}
        >
          {name}
        </span>
        <span
          style={{
            minWidth: 40,
            height: 40,
            padding: "0 12px",
            borderRadius: 9999,
            backgroundColor: "#F44336",
            color: "#FFFFFF",
            fontSize: 24,
            fontWeight: 700,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
          }}
        >
          {unread}
        </span>
      </div>
      <div
        style={{
          display: "flex",
          flexDirection: "column",
          gap: 16,
          padding: 28,
          fontSize: 30,
          lineHeight: 1.3,
          color: "#9E9E9E",
        }}
      >
        {lines.map((line) => (
          <span key={line}>{line}</span>
        ))}
      </div>
    </div>
  );
};

export const Chaos = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  return (
    <AbsoluteFill>
      <Interactive.Div
        name="Windows"
        style={{
          position: "absolute",
          inset: 0,
          opacity: interpolate(frame, [6 * fps, 6.5 * fps], [1, 0.18], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
        }}
      >
        <AppWindow
          name="GitHub"
          unread={3}
          left={100}
          top={100}
          opensAt={0.3}
          lines={["acme/checkout#15: changes requested", "CI failed on main"]}
        />
        <AppWindow
          name="Slack"
          unread={5}
          left={1200}
          top={120}
          opensAt={0.85}
          lines={["#deploys  Ana: can you look?", "3 unread DMs"]}
        />
        <AppWindow
          name="Jira"
          unread={2}
          left={650}
          top={60}
          opensAt={1.4}
          lines={["PROJ-7 assigned to you", "PROJ-9 new comment"]}
        />
        <AppWindow
          name="Datadog"
          unread={1}
          left={160}
          top={620}
          opensAt={1.95}
          lines={["p95 latency above 800ms", "checkout · Alert"]}
        />
        <AppWindow
          name="Linear"
          unread={4}
          left={1140}
          top={640}
          opensAt={2.5}
          lines={["ENG-12 assigned to you", "ENG-4 moved to In Review"]}
        />
        <AppWindow
          name="Discord"
          unread={2}
          left={420}
          top={360}
          opensAt={3.05}
          lines={["@you in #help: is this a bug?"]}
        />
        <AppWindow
          name="Todoist"
          unread={1}
          left={900}
          top={400}
          opensAt={3.6}
          lines={["Renew the certificate, due today"]}
        />
        <svg
          width={48}
          height={48}
          viewBox="0 0 24 24"
          style={{
            position: "absolute",
            left: interpolate(
              frame,
              [1 * fps, 1.6 * fps, 2.2 * fps, 2.8 * fps, 3.4 * fps, 4 * fps, 4.6 * fps, 5.2 * fps],
              [420, 1500, 950, 460, 1440, 720, 1200, 520],
              {
                extrapolateLeft: "clamp",
                extrapolateRight: "clamp",
                easing: Easing.bezier(0.65, 0, 0.35, 1),
              },
            ),
            top: interpolate(
              frame,
              [1 * fps, 1.6 * fps, 2.2 * fps, 2.8 * fps, 3.4 * fps, 4 * fps, 4.6 * fps, 5.2 * fps],
              [200, 220, 160, 720, 740, 460, 500, 220],
              {
                extrapolateLeft: "clamp",
                extrapolateRight: "clamp",
                easing: Easing.bezier(0.65, 0, 0.35, 1),
              },
            ),
            opacity: interpolate(frame, [0.8 * fps, 1 * fps], [0, 1], {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
            }),
          }}
        >
          <path
            d="M4 2l16 10-7 1.5L9.5 21z"
            fill="#FFFFFF"
            stroke="#000000"
            strokeWidth={1.2}
            strokeLinejoin="round"
          />
        </svg>
      </Interactive.Div>
      <AbsoluteFill
        style={{
          justifyContent: "center",
          padding: "0 160px",
          gap: 32,
          fontFamily: "Inter",
        }}
      >
        <Interactive.Div
          name="Headline"
          style={{
            fontWeight: 700,
            fontSize: 150,
            lineHeight: 1.05,
            letterSpacing: "-0.03em",
            color: "#FFFFFF",
            opacity: interpolate(frame, [6.3 * fps, 6.9 * fps], [0, 1], {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
            }),
            translate: interpolate(
              frame,
              [6.3 * fps, 7.1 * fps],
              ["0px 40px", "0px 0px"],
              {
                extrapolateLeft: "clamp",
                extrapolateRight: "clamp",
                easing: Easing.bezier(0.16, 1, 0.3, 1),
              },
            ),
          }}
        >
          Seven tabs to watch
        </Interactive.Div>
        <Interactive.Div
          name="Subhead"
          style={{
            fontSize: 78,
            lineHeight: 1.2,
            color: "#9E9E9E",
            opacity: interpolate(frame, [7.4 * fps, 8 * fps], [0, 1], {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
            }),
          }}
        >
          and every check pulls you out of the code.
        </Interactive.Div>
      </AbsoluteFill>
    </AbsoluteFill>
  );
};

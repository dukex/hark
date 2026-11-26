import type { CSSProperties } from "react";
import { AppWindow } from "./AppWindow";

type Size = {
  readonly width: number;
  readonly height: number;
  readonly style?: CSSProperties;
};

const Avatar = ({
  letter,
  color,
  size = 40,
  radius = 9999,
}: {
  readonly letter: string;
  readonly color: string;
  readonly size?: number;
  readonly radius?: number;
}) => (
  <span
    style={{
      flexShrink: 0,
      width: size,
      height: size,
      borderRadius: radius,
      backgroundColor: color,
      color: "#FFFFFF",
      fontSize: size * 0.45,
      fontWeight: 700,
      display: "flex",
      alignItems: "center",
      justifyContent: "center",
    }}
  >
    {letter}
  </span>
);

export const GitHubPR = ({
  review = 1,
  ...size
}: Size & { readonly review?: number }) => (
  <AppWindow
    {...size}
    url="github.com/acme/checkout/pull/15"
    chrome="#161B22"
    background="#0D1117"
  >
    <div
      style={{
        flex: 1,
        display: "flex",
        flexDirection: "column",
        gap: 18,
        padding: 28,
        color: "#E6EDF3",
      }}
    >
      <span style={{ fontSize: 30, fontWeight: 600, lineHeight: 1.25 }}>
        Move refund flow to the new payments client{" "}
        <span style={{ color: "#7D8590", fontWeight: 400 }}>#15</span>
      </span>
      <span
        style={{
          display: "flex",
          alignItems: "center",
          gap: 12,
          fontSize: 19,
          color: "#7D8590",
        }}
      >
        <span
          style={{
            padding: "4px 14px",
            borderRadius: 9999,
            backgroundColor: "#238636",
            color: "#FFFFFF",
            fontWeight: 600,
          }}
        >
          Open
        </span>
        dukex wants to merge 4 commits into main
      </span>
      <div
        style={{
          display: "flex",
          gap: 14,
          paddingTop: 18,
          borderTop: "1px solid #30363D",
          opacity: review,
          translate: `0px ${(1 - review) * 24}px`,
        }}
      >
        <Avatar letter="C" color="#BF4B8A" />
        <div
          style={{ flex: 1, display: "flex", flexDirection: "column", gap: 10 }}
        >
          <span style={{ fontSize: 22 }}>
            <b>carol</b>{" "}
            <span style={{ color: "#F85149" }}>requested changes</span>{" "}
            <span style={{ color: "#7D8590" }}>just now</span>
          </span>
          <span
            style={{
              padding: "14px 18px",
              border: "1px solid #30363D",
              borderRadius: 8,
              fontSize: 22,
            }}
          >
            Please move the migration to its own PR.
          </span>
          <span style={{ fontSize: 18, color: "#7D8590" }}>
            2 inline comments on db/migrations/0042_refunds.sql
          </span>
        </div>
      </div>
    </div>
  </AppWindow>
);

export const SlackChannel = (size: Size) => (
  <AppWindow {...size} title="acme · Slack" chrome="#2C0F2D" background="#1A1D21">
    <div
      style={{
        width: 190,
        flexShrink: 0,
        display: "flex",
        flexDirection: "column",
        gap: 10,
        padding: "20px 18px",
        backgroundColor: "#3F0E40",
        color: "#CFC3CF",
        fontSize: 19,
      }}
    >
      <span style={{ color: "#FFFFFF", fontWeight: 700, fontSize: 22 }}>
        acme
      </span>
      <span># general</span>
      <span
        style={{
          margin: "0 -18px",
          padding: "4px 18px",
          backgroundColor: "#1164A3",
          color: "#FFFFFF",
          fontWeight: 700,
        }}
      >
        # deploys
      </span>
      <span># random</span>
      <span style={{ color: "#FFFFFF", fontWeight: 700 }}># incidents</span>
    </div>
    <div
      style={{
        flex: 1,
        display: "flex",
        flexDirection: "column",
        color: "#D1D2D3",
      }}
    >
      <span
        style={{
          padding: "16px 24px",
          borderBottom: "1px solid #2E3136",
          fontSize: 22,
          fontWeight: 700,
          color: "#FFFFFF",
        }}
      >
        # deploys
      </span>
      <div style={{ display: "flex", gap: 14, padding: 24 }}>
        <Avatar letter="A" color="#E8912D" radius={8} />
        <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
          <span style={{ fontSize: 20 }}>
            <b style={{ color: "#FFFFFF" }}>Ana</b>{" "}
            <span style={{ color: "#ABABAD", fontSize: 16 }}>10:31 AM</span>
          </span>
          <span style={{ fontSize: 21, lineHeight: 1.35 }}>
            <span
              style={{
                padding: "0 4px",
                borderRadius: 4,
                backgroundColor: "rgba(242, 199, 68, 0.2)",
                color: "#F2C744",
              }}
            >
              @you
            </span>{" "}
            can you look at the rollout? checkout is at 40%
          </span>
        </div>
      </div>
    </div>
  </AppWindow>
);

export const JiraIssue = (size: Size) => (
  <AppWindow
    {...size}
    url="example.atlassian.net/browse/PROJ-7"
    chrome="#161A1D"
    background="#1D2125"
  >
    <div
      style={{
        flex: 1,
        display: "flex",
        flexDirection: "column",
        gap: 16,
        padding: 28,
        color: "#B6C2CF",
      }}
    >
      <span style={{ fontSize: 17, color: "#8C9BAB" }}>
        Projects / Payments / PROJ-7
      </span>
      <span style={{ fontSize: 28, fontWeight: 600, color: "#DEE4EA" }}>
        Refund webhook retries twice
      </span>
      <span
        style={{
          alignSelf: "flex-start",
          padding: "4px 10px",
          borderRadius: 4,
          backgroundColor: "#2C333A",
          fontSize: 16,
          fontWeight: 700,
        }}
      >
        TO DO
      </span>
      <span
        style={{ display: "flex", alignItems: "center", gap: 12, fontSize: 19 }}
      >
        <span style={{ width: 100, color: "#8C9BAB" }}>Assignee</span>
        <Avatar letter="Y" color="#0C66E4" size={32} />
        You
      </span>
      <span style={{ fontSize: 18, color: "#579DFF" }}>
        Carlos assigned this to you just now
      </span>
    </div>
  </AppWindow>
);

export const DatadogMonitor = (size: Size) => (
  <AppWindow
    {...size}
    url="app.datadoghq.com/monitors/42"
    chrome="#1A1A1E"
    background="#202024"
  >
    <div style={{ flex: 1, display: "flex", flexDirection: "column" }}>
      <span style={{ height: 8, backgroundColor: "#632CA6" }} />
      <div
        style={{
          flex: 1,
          display: "flex",
          flexDirection: "column",
          gap: 14,
          padding: 26,
          color: "#E2E3E5",
        }}
      >
        <span style={{ display: "flex", alignItems: "center", gap: 14 }}>
          <span
            style={{
              padding: "4px 12px",
              borderRadius: 4,
              backgroundColor: "#EB364B",
              color: "#FFFFFF",
              fontSize: 16,
              fontWeight: 700,
            }}
          >
            ALERT
          </span>
          <span style={{ fontSize: 22, fontWeight: 600 }}>
            p95 latency above 800ms
          </span>
        </span>
        <span style={{ fontSize: 17, color: "#9FA1A7" }}>
          service:checkout · env:prod
        </span>
        <svg viewBox="0 0 400 120" style={{ width: "100%", flex: 1 }}>
          <line
            x1={0}
            y1={40}
            x2={400}
            y2={40}
            stroke="#EB364B"
            strokeWidth={2}
            strokeDasharray="6 6"
          />
          <polyline
            points="0,100 40,96 80,98 120,92 160,95 200,88 240,80 280,62 310,44 340,30 370,22 400,18"
            fill="none"
            stroke="#8A6FDF"
            strokeWidth={3}
          />
        </svg>
      </div>
    </div>
  </AppWindow>
);

export const LinearInbox = (size: Size) => (
  <AppWindow
    {...size}
    url="linear.app/acme/inbox"
    chrome="#101114"
    background="#16171B"
  >
    <div
      style={{
        flex: 1,
        display: "flex",
        flexDirection: "column",
        color: "#E3E4E8",
      }}
    >
      <span
        style={{
          padding: "18px 24px",
          borderBottom: "1px solid #26272D",
          fontSize: 22,
          fontWeight: 600,
        }}
      >
        Inbox
      </span>
      {[
        ["ENG-12", "Retry failed refunds", "Assigned to you"],
        ["ENG-4", "Payments client timeouts", "Moved to In Review"],
        ["ENG-9", "Refund email copy", "Mentioned you"],
      ].map(([id, title, reason]) => (
        <div
          key={id}
          style={{
            display: "flex",
            alignItems: "center",
            gap: 14,
            padding: "14px 24px",
            borderBottom: "1px solid #1F2026",
          }}
        >
          <span
            style={{
              width: 10,
              height: 10,
              borderRadius: 9999,
              backgroundColor: "#5E6AD2",
            }}
          />
          <span style={{ display: "flex", flexDirection: "column", gap: 4 }}>
            <span style={{ fontSize: 19 }}>
              <span style={{ color: "#8A8F98" }}>{id}</span> {title}
            </span>
            <span style={{ fontSize: 16, color: "#8A8F98" }}>{reason}</span>
          </span>
        </div>
      ))}
    </div>
  </AppWindow>
);

export const DiscordChannel = (size: Size) => (
  <AppWindow {...size} title="Discord" chrome="#1E1F22" background="#313338">
    <div
      style={{
        width: 72,
        flexShrink: 0,
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        gap: 12,
        paddingTop: 14,
        backgroundColor: "#1E1F22",
      }}
    >
      {["#5865F2", "#3BA55D", "#EB459E"].map((color) => (
        <span
          key={color}
          style={{
            width: 46,
            height: 46,
            borderRadius: 16,
            backgroundColor: color,
          }}
        />
      ))}
    </div>
    <div
      style={{
        flex: 1,
        display: "flex",
        flexDirection: "column",
        color: "#DBDEE1",
      }}
    >
      <span
        style={{
          padding: "16px 22px",
          borderBottom: "1px solid #26282C",
          fontSize: 21,
          fontWeight: 700,
          color: "#F2F3F5",
        }}
      >
        # help
      </span>
      <div
        style={{
          display: "flex",
          gap: 14,
          margin: "18px 0",
          padding: "10px 22px",
          backgroundColor: "rgba(250, 168, 26, 0.08)",
          borderLeft: "3px solid #F0B232",
        }}
      >
        <Avatar letter="B" color="#5865F2" />
        <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
          <span style={{ fontSize: 19 }}>
            <b style={{ color: "#F2F3F5" }}>ben</b>{" "}
            <span style={{ color: "#949BA4", fontSize: 15 }}>
              Today at 10:40
            </span>
          </span>
          <span style={{ fontSize: 20 }}>
            <span
              style={{
                padding: "0 3px",
                borderRadius: 3,
                backgroundColor: "rgba(88, 101, 242, 0.3)",
                color: "#C9CDFB",
              }}
            >
              @you
            </span>{" "}
            is this a bug in 0.8?
          </span>
        </div>
      </div>
    </div>
  </AppWindow>
);

export const TodoistToday = (size: Size) => (
  <AppWindow
    {...size}
    url="app.todoist.com/app/today"
    chrome="#1A1A1A"
    background="#1E1E1E"
  >
    <div
      style={{
        flex: 1,
        display: "flex",
        flexDirection: "column",
        gap: 6,
        padding: 26,
        color: "#E6E6E6",
      }}
    >
      <span style={{ fontSize: 26, fontWeight: 700, marginBottom: 10 }}>
        Today
      </span>
      {["Renew the certificate", "Reply to the refund RFC", "Book the dentist"].map(
        (task) => (
          <span
            key={task}
            style={{
              display: "flex",
              alignItems: "center",
              gap: 14,
              padding: "12px 0",
              borderBottom: "1px solid #2B2B2B",
              fontSize: 20,
            }}
          >
            <span
              style={{
                width: 20,
                height: 20,
                borderRadius: 9999,
                border: "2px solid #DC4C3E",
              }}
            />
            <span style={{ flex: 1 }}>{task}</span>
            <span style={{ fontSize: 15, color: "#DC4C3E" }}>Today</span>
          </span>
        ),
      )}
    </div>
  </AppWindow>
);

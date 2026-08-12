import { IconMail } from "@tabler/icons-react";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import StatCard from "@/pages/(panel)/dashboard/StatCard";

const renderCard = (props: Partial<Parameters<typeof StatCard>[0]> = {}) =>
  render(
    <MemoryRouter>
      <StatCard
        icon={IconMail}
        label="Unread messages"
        value={3}
        to="/contact"
        accent="primary"
        {...props}
      />
    </MemoryRouter>,
  );

describe("StatCard", () => {
  it("renders its label and value", () => {
    renderCard();
    expect(screen.getByText("Unread messages")).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
  });

  it("links to the module it counts", () => {
    renderCard();
    expect(screen.getByRole("link")).toHaveAttribute("href", "/contact");
  });

  // A count of zero is a real answer, not a missing one -- rendering nothing
  // would leave the card looking broken on a fresh install.
  it("renders a zero value rather than blanking out", () => {
    renderCard({ value: 0 });
    expect(screen.getByText("0")).toBeInTheDocument();
  });

  it("shows a skeleton instead of a stale number while loading", () => {
    renderCard({ loading: true });
    expect(screen.queryByText("3")).not.toBeInTheDocument();
  });

  it("hides the hint while loading", () => {
    renderCard({ hint: "12 in total", loading: true });
    expect(screen.queryByText("12 in total")).not.toBeInTheDocument();
  });

  it("shows the hint once loaded", () => {
    renderCard({ hint: "12 in total" });
    expect(screen.getByText("12 in total")).toBeInTheDocument();
  });

  // The background icon is decoration; the label and number carry the meaning.
  it("hides the decorative icon from assistive technology", () => {
    const { container } = renderCard();
    const icon = container.querySelector("svg");
    expect(icon).toHaveAttribute("aria-hidden");
  });

  // Without clipping, the icon's deliberate overhang would spill outside the
  // card instead of being cut off by its edge.
  it("clips the overflowing background icon", () => {
    const { container } = renderCard();
    expect(container.querySelector(".overflow-clip")).toBeInTheDocument();
  });
});

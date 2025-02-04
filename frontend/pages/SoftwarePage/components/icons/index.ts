import AcrobatReader from "./AcrobatReader";
import ChromeApp from "./ChromeApp";
import Excel from "./Excel";
import Extension from "./Extension";
import Firefox from "./Firefox";
import MacApp from "./MacApp";
import Package from "./Package";
import Safari from "./Safari";
import Slack from "./Slack";
import Teams from "./Teams";
import VisualStudioCode from "./VisualStudioCode";
import WindowsApp from "./WindowsApp";
import Word from "./Word";
import Zoom from "./Zoom";

// SOFTWARE_NAME_TO_ICON_MAP list "special" applications that have a defined
// icon for them, keys refer to application names, and are intended to be fuzzy
// matched in the application logic.
export const SOFTWARE_NAME_TO_ICON_MAP = {
  "adobe acrobat reader": AcrobatReader,
  "google chrome": ChromeApp,
  "microsoft excel": Excel,
  firefox: Firefox,
  package: Package,
  safari: Safari,
  slack: Slack,
  "microsoft teams": Teams,
  "visual studio code": VisualStudioCode,
  "microsoft word": Word,
  zoom: Zoom,
} as const;

// SOFTWARE_SOURCE_TO_ICON_MAP maps different software sources to a defined
// icon.
export const SOFTWARE_SOURCE_TO_ICON_MAP = {
  package: Package,
  apt_sources: Package,
  deb_packages: Package,
  rpm_packages: Package,
  yum_sources: Package,
  npm_packages: Package,
  atom_packages: Package,
  python_packages: Package,
  homebrew_packages: Package,
  apps: MacApp,
  programs: WindowsApp,
  chrome_extensions: Extension,
  safari_extensions: Extension,
  firefox_addons: Extension,
  ie_extensions: Extension,
  chocolatey_packages: Package,
  pkg_packages: Package,
} as const;

export const SOFTWARE_ICON_SIZES: Record<string, string> = {
  medium: "24",
  large: "96",
} as const;

export type SoftwareIconSizes = keyof typeof SOFTWARE_ICON_SIZES;

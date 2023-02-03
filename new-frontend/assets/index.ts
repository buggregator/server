import 'highlight.js/lib/common';
import Sfdump from "~/vendor/dumper";

declare global {
  interface Window { Sfdump: () => void; }
}

window.Sfdump = Sfdump

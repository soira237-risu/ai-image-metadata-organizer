import { DetailPanel } from "./components/DetailPanel";
import { FilterSidebar } from "./components/FilterSidebar";
import { MovePanel } from "./components/MovePanel";
import { ResultsList } from "./components/ResultsList";
import { StatusBar } from "./components/StatusBar";
import { Toolbar } from "./components/Toolbar";
import { useLibraryController } from "./hooks/useLibraryController";
import "./styles.css";

export default function App() {
  const controller = useLibraryController();
  return (
    <main className="app-shell" onDragOver={(event) => event.preventDefault()} onDrop={(event) => event.preventDefault()}>
      <Toolbar controller={controller} />
      <section className="workspace">
        <FilterSidebar controller={controller} />
        <ResultsList controller={controller} />
        <DetailPanel detail={controller.detail} selected={controller.selected} />
      </section>
      <MovePanel controller={controller} />
      <StatusBar controller={controller} />
    </main>
  );
}

use std::sync::{Arc, Mutex};
use tauri::Manager;
use tauri_plugin_shell::ShellExt;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
  tauri::Builder::default()
    .plugin(tauri_plugin_dialog::init())
    .plugin(tauri_plugin_shell::init())
    .setup(|app| {
      let data_dir = if cfg!(debug_assertions) {
        let exe = std::env::current_exe().expect("failed to get exe path");
        let project_root = exe.parent().unwrap()
          .parent().unwrap()  // target/<profile>
          .parent().unwrap()  // target
          .parent().unwrap()  // src-tauri
          .parent().unwrap()  // frontend
          .parent().unwrap()  // project root
          .join("backend").join("data");
        std::fs::create_dir_all(&project_root).ok();
        project_root
      } else {
        let exe = std::env::current_exe().expect("failed to get exe path");
        let app_dir = exe.parent().expect("failed to get app directory");
        let dir = app_dir.join("data");
        std::fs::create_dir_all(&dir).ok();
        dir
      };

      let sidecar = app.shell().sidecar("owl-backend").expect("failed to create sidecar command");
      let (_rx, child) = sidecar
        .args(["--port", "3721", "--data-dir", data_dir.to_str().unwrap_or("data")])
        .spawn()
        .expect("failed to spawn Go backend sidecar");

      app.manage(Arc::new(Mutex::new(Some(child))));

      if cfg!(debug_assertions) {
        app.handle().plugin(
          tauri_plugin_log::Builder::default()
            .level(log::LevelFilter::Info)
            .build(),
        )?;
      }

      Ok(())
    })
    .on_window_event(|window, event| {
      if let tauri::WindowEvent::CloseRequested { .. } = event {
        if let Some(state) = window.try_state::<Arc<Mutex<Option<tauri_plugin_shell::process::CommandChild>>>>() {
          if let Ok(mut guard) = state.lock() {
            if let Some(c) = guard.take() {
              c.kill().ok();
            }
          }
        }
      }
    })
    .run(tauri::generate_context!())
    .expect("error while running tauri application");
}

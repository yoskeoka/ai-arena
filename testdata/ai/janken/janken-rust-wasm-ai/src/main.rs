use std::io::{self, BufRead, Write};

fn main() -> io::Result<()> {
    let stdin = io::stdin();
    let mut stdout = io::stdout();

    for line in stdin.lock().lines() {
        let line = line?;
        let Some(method) = extract_string_field(&line, "method") else {
            continue;
        };
        let Some(id) = extract_string_field(&line, "id") else {
            continue;
        };

        match method.as_str() {
            "init" => {
                eprintln!("janken-rust-wasm-ai init");
                writeln!(
                    stdout,
                    "{{\"jsonrpc\":\"2.0\",\"id\":\"{}\",\"result\":{{\"ready\":true}}}}",
                    escape_json_string(&id),
                )?;
                stdout.flush()?;
            }
            "turn" => {
                let round = extract_round(&line).unwrap_or(0);
                eprintln!("janken-rust-wasm-ai turn {}", round);
                writeln!(
                    stdout,
                    "{{\"jsonrpc\":\"2.0\",\"id\":\"{}\",\"result\":{{\"action\":\"paper\"}}}}",
                    escape_json_string(&id),
                )?;
                stdout.flush()?;
            }
            "game_over" => {
                eprintln!("janken-rust-wasm-ai game_over");
                writeln!(
                    stdout,
                    "{{\"jsonrpc\":\"2.0\",\"id\":\"{}\",\"result\":{{\"ack\":true}}}}",
                    escape_json_string(&id),
                )?;
                stdout.flush()?;
                return Ok(());
            }
            _ => {}
        }
    }

    Ok(())
}

fn extract_string_field(line: &str, field: &str) -> Option<String> {
    let needle = format!("\"{}\"", field);
    let start = line.find(&needle)?;
    let after_field = &line[start + needle.len()..];
    let colon = after_field.find(':')?;
    let after_colon = after_field[colon + 1..].trim_start();
    let quoted = after_colon.strip_prefix('"')?;
    let end = quoted.find('"')?;
    Some(quoted[..end].to_string())
}

fn extract_round(line: &str) -> Option<u32> {
    let needle = "\"round\"";
    let start = line.find(needle)?;
    let after_field = &line[start + needle.len()..];
    let colon = after_field.find(':')?;
    let digits = after_field[colon + 1..]
        .trim_start()
        .chars()
        .take_while(|c| c.is_ascii_digit())
        .collect::<String>();
    digits.parse().ok()
}

fn escape_json_string(value: &str) -> String {
    value.replace('\\', "\\\\").replace('"', "\\\"")
}

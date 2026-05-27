import * as path from 'path';
import * as cp from 'child_process';
import { workspace, window, ExtensionContext, commands, OutputChannel, Uri } from 'vscode';
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind,
} from 'vscode-languageclient/node';

let client: LanguageClient;
let outputChannel: OutputChannel;

export function activate(context: ExtensionContext) {
  outputChannel = window.createOutputChannel('Carv');

  const serverPath = workspace.getConfiguration('carv.languageServer').get<string>('path', 'carv');

  const serverOptions: ServerOptions = {
    command: serverPath,
    args: ['lsp'],
    transport: TransportKind.stdio,
  };

  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: 'file', language: 'carv' }],
    synchronize: {
      fileEvents: workspace.createFileSystemWatcher('**/*.carv'),
    },
  };

  client = new LanguageClient(
    'carv-lsp',
    'Carv Language Server',
    serverOptions,
    clientOptions
  );

  client.start();

  // Register the `carv.runFile` command to execute the current .carv file
  const runDisposable = commands.registerCommand('carv.runFile', async () => {
    const editor = window.activeTextEditor;
    if (!editor) {
      window.showErrorMessage('No active editor');
      return;
    }
    const doc = editor.document;
    if (doc.languageId !== 'carv') {
      window.showErrorMessage('Not a Carv file');
      return;
    }
    const filePath = doc.uri.fsPath;
    const binPath = workspace.getConfiguration('carv.languageServer').get<string>('path', 'carv');

    outputChannel.clear();
    outputChannel.show();
    outputChannel.appendLine(`$ ${binPath} run "${filePath}"`);

    try {
      const result = cp.execFileSync(binPath, ['run', filePath], {
        encoding: 'utf-8',
        timeout: 30000,
      });
      outputChannel.append(result);
    } catch (err: any) {
      if (err.stdout) {
        outputChannel.append(err.stdout);
      }
      if (err.stderr) {
        outputChannel.append(err.stderr);
      }
      if (err.status !== undefined) {
        outputChannel.appendLine(`\n[exited with code ${err.status}]`);
      }
    }
  });

  context.subscriptions.push(runDisposable, outputChannel);
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}

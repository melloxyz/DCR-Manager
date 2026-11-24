# Roadmap oficial

Escopo de produto fornecido pelo mantenedor em 2026-11-24. As etapas T1-T10 são entregas internas da v1.3.0, não substituem as versões de produto. O primeiro snapshot Git real é v1.2.1; não existem tags históricas inventadas.

## v1.1.0 - Fundação estrutural
- Arquitetura modular: implementada.
- Instalação permanente: implementada.
- Gerenciamento de processos: implementado.
- Diagnóstico e reparo seletivo: implementado.
- Testes automatizados: implementados.
- Configuração de CI/CD: CI configurada; execução hospedada e CD ainda pendentes.

## v1.2.0 - Gerenciamento avançado
- Detectar instalações globais e locais: implementado.
- Auditar configurações MCP dos agentes: implementado em locais conhecidos, leitura apenas; TOML opcional.
- Identificar processos órfãos: avaliação de pais ausentes/reutilizados, não prova de processo órfão.
- Rollback automatizado: implementado com tentativa de recuperação e relato de falhas.
- Gerenciar backups e retenção: inventário, restauração e revisão de idade; sem exclusão automática.
- Verificar compatibilidade Node.js/npm: versões e requisitos mínimos simples; faixas complexas pendentes.

## v1.3.0 - Experiência de uso
- Interface com navegação por teclado: implementada em Go/Charm v2, Graphite Dark.
- Inglês: idioma único na v1.3.0, por decisão do mantenedor. Internacionalização adiada para avaliação futura.
- Painel de saúde do sistema: implementado, diagnósticos locais.
- Assistente de resolução de problemas: implementado; regras locais e recomendações sem execução automática.
- Instalação inicial guiada: implementada; pré-requisitos, navegação para instalação confirmada e conexão, com opção de concluir ou pular.
- Gerenciamento de inicialização com Windows: implementado; registro por usuário, opt-in e reversível; validação em login real pendente.

### Plano interno atualizado
| Etapa | Entrega | Estado |
|---|---|---|
| T1 | Fundação Go e arquitetura | Implementada |
| T2 | Design system Graphite Dark | Implementada |
| T3 | Shell e navegação | Implementada |
| T4 | Dashboard com dados reais | Implementada |
| T5 | Controle nativo de processos Windows | Implementada; reconexão real pendente |
| T6 | Pacotes e cache | Implementada; npm real isolado pendente |
| T7 | Backups, recuperação, inventário de estágios e retenção | Implementada; restauração real isolada pendente |
| T8 | Auditoria MCP, logs e assistente de resolução de problemas | Implementada; validação de UX pendente |
| T9 | Configurações persistentes, onboarding em inglês e inicialização com Windows (opt-in e reversível) | Implementada; aceitação visual e login real pendentes |
| T10 | Estabilização, regressões e empacotamento local | RC1 preparada; aceite interativo e validação real pendentes |

## v1.4.0 - Distribuição
- Avaliar migração para Go: antecipada e aprovada na v1.3.0; concluir avaliação de dependências remanescentes e portabilidade.
- Executável portátil: pendente; preview ainda requer scripts adjacentes.
- Releases automatizadas: pendente.
- Licença open-source: pendente de decisão do mantenedor.
- Documentação pública: pendente.
- Testes em máquinas Windows limpas: pendente.

Antes de publicar: testar o pacote real, autenticação/reconexão, backup/restore, instalações limpas, CI hospedada, segurança, licença, marca upstream e artefatos verificáveis. Não publicar dados locais. v1.3.0 estável não implica release pública da v1.4.0.

## v2.0.0 - Gerenciador completo
- Gerenciar integrações com agentes.
- Migrar instalações antigas.
- Diagnóstico avançado de conectividade.
- Gerenciar sessões e logs.
- Avaliar suporte Linux e macOS.

Todos os itens v2.0.0 permanecem pendentes; inventários e diagnósticos básicos existentes não equivalem a esses recursos completos.

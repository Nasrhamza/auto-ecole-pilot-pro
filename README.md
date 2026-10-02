# AutoÉcole Pilot Pro v1

Application Windows locale pour un moniteur indépendant : élèves, heures, agenda, examens et paiements.

## Mise à jour du 12 septembre 2026

- Interface lisible, bordures sobres et navigation centrée sur le moniteur.
- Défilement natif, tableaux adaptés à la largeur et paramètres en trois rubriques.
- Planning : changement de semaine corrigé, début/fin des séances, proposition de créneau libre.
- Planning semaine/mois : sélection du dimanche corrigée ; aperçu des élèves et horaires sur les jours du calendrier.
- Nouvelle séance → Programmer plusieurs séances : même jour chaque semaine, plusieurs jours de la semaine/du mois, ou jusqu’à une date. Choisir les jours puis vérifier la liste avant enregistrement. La période commence à la date de la première séance ; les conflits connus sont contrôlés avant la création.
- Compteurs distincts : heures réalisées, restantes et déjà planifiées.
- Priorités des élèves enregistrées dans les données, indépendamment du port de l’application.
- Dossier : tous les fichiers sont accessibles, même plusieurs documents de même catégorie.
- Échec en conduite : résultat et rattrapage/réinscription enregistrés ensemble ; demi-heures acceptées.
- Contrôles des chevauchements et de l’heure de fin, protection contre les doubles soumissions.

Fermer l’application avant de relancer le Setup. Les données et la licence existantes ne sont pas supprimées par cette mise à jour. Faire une sauvegarde depuis Paramètres avant installation.

## Installation client

Envoyer uniquement `A-ENVOYER-AU-CLIENT/AutoEcole-Pilot-Pro-v1-Setup.exe`.
Le Setup installe l'application, son raccourci Bureau et Microsoft WebView2 si nécessaire.

Au premier démarrage, le client transmet le code machine affiché. La clé produit est créée avec l'outil privé `owner-tools/AutoEcole-Pilot-License-Generator.exe`. Une clé n'est valable que sur le PC concerné et peut être permanente ou limitée à 30 jours, 3 mois, 1 an ou 2 ans.

## Fonctions

- Candidats, forfaits et soldes
- Planning semaine code/conduite avec blocage des conflits
- Mon véhicule et ses échéances
- Dossiers numériques PDF/images et certificats médicaux
- Forfaits d'heures, progression et présences
- Examens, résultats et certificats de réussite PDF
- Factures clients, bons de réception et bons de rendez-vous PDF
- Paiements, dépenses, rapports par période et export Excel/CSV
- Historique entretien, vidange, réparation et carburant des véhicules
- Absences et performances des moniteurs
- Alertes configurables assurance/visite/entretien/examens/dossiers
- Rappels WhatsApp prêts à envoyer
- Identité, logo, couleur, sauvegarde et restauration locale
- Compte du moniteur et traçabilité des opérations
- Licence liée à la machine, permanente ou limitée dans le temps

Le dossier `owner-tools` et le code source ne doivent jamais être envoyés au client.
